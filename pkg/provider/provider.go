package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog"

	"github.com/ProtonMail/gopenpgp/v2/crypto"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/checksum"
	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/cosign"
	"github.com/ekristen/distillery/pkg/osconfig"
	"github.com/ekristen/distillery/pkg/score"
)

const (
	VersionLatest = "latest"
	ChecksumType  = "checksum"

	SignatureTypeNone     = "none"
	SignatureTypeFile     = "file"
	SignatureTypeChecksum = "checksum"
)

type Options struct {
	OS       string
	Arch     string
	Config   *config.Config
	Logger   zerolog.Logger
	Settings map[string]interface{}
}

type Provider struct {
	Options   *Options
	OSConfig  *osconfig.OS
	Assets    []asset.IAsset
	Binary    asset.IAsset
	Signature asset.IAsset
	Checksum  asset.IAsset
	Key       asset.IAsset

	ChecksumType  string
	SignatureType string

	Logger zerolog.Logger
}

func (p *Provider) GetOS() string {
	return p.Options.OS
}

func (p *Provider) GetArch() string {
	return p.Options.Arch
}

func (p *Provider) GetVersion() string {
	return "not-implemented"
}

func (p *Provider) GetOptions() *Options {
	return p.Options
}

// CommonRun - common run logic for all sources that includes download, extract, install and cleanup
func (p *Provider) CommonRun(ctx context.Context) error {
	if err := p.Download(ctx); err != nil {
		return err
	}

	defer func(s *Provider) {
		err := s.Cleanup()
		if err != nil {
			p.Logger.Error().Err(err).Msg("unable to cleanup")
		}
	}(p)

	if err := p.Verify(); err != nil {
		return err
	}

	if err := p.Extract(); err != nil {
		return err
	}

	if err := p.Install(); err != nil {
		return err
	}

	return nil
}

func (p *Provider) discoverBinary(names []string, version string) error { //nolint:gocyclo,funlen
	p.Logger.Trace().Msgf("names: %v", names)

	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	p.Logger.Trace().Msgf("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		p.Logger.Trace().Msgf("discover: type: %d, files: %d", k, len(v))
	}

	highEnoughScore := false

	// Note: first pass we want to look for just binaries, archives and unknowns and score and sort them
	for k, v := range fileScoring {
		if k != asset.Binary && k != asset.Unknown && k != asset.Archive {
			continue
		}

		detectedOS := p.OSConfig.GetOS()
		arch := p.OSConfig.GetArchitectures()
		ext := p.OSConfig.GetExtensions()

		if _, ok := fileScored[k]; !ok {
			fileScored[k] = []score.Sorted{}
		}

		terms := names
		for _, name := range p.OSConfig.GetLibraryNames() {
			terms = append(terms, name)
		}

		weightedTerms := map[string]int{
			"source": -20, // as in source.tar.gz
		}

		fileScored[k] = score.Score(v, &score.Options{
			OS:                detectedOS,
			Arch:              arch,
			Extensions:        ext,
			Terms:             terms,
			WeightedTerms:     weightedTerms,
			Versions:          []string{version},
			InvalidOS:         p.OSConfig.InvalidOS(),
			InvalidArch:       p.OSConfig.InvalidArchitectures(),
			InvalidExtensions: []string{".zst"},
		})

		if len(fileScored[k]) > 0 {
			for _, vv := range fileScored[k] {
				if vv.Value >= 40 {
					p.Logger.Debug().Msgf("HIGH SCORE")
					highEnoughScore = true
				}
				p.Logger.Debug().Msgf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	if !highEnoughScore && !p.Options.Settings["no-score-check"].(bool) {
		p.Logger.Error().Msg("no matching asset found, score too low")
		for _, t := range []asset.Type{asset.Binary, asset.Unknown, asset.Archive} {
			for _, v := range fileScored[t] {
				if v.Value < 40 {
					p.Logger.Error().Msgf("closest matching: %s (%d) (threshold: 40) -- override with --no-score-check", v.Key, v.Value)
					return errors.New("no matching asset found, score too low")
				}
			}
		}

		return errors.New("no matching asset found, score too low")
	}

	// Note: we want to look for the best binary by looking at binaries, archives and unknowns
	for _, t := range []asset.Type{asset.Unknown, asset.Binary, asset.Archive} {
		if p.Binary != nil {
			break
		}

		if len(fileScored[t]) > 0 {
			p.Logger.Trace().
				Str("filename", fileScored[t][0].Key).
				Int("score", fileScored[t][0].Value).
				Str("filetype", t.String()).
				Msgf("top scored (%d): %s (%d)", t, fileScored[t][0].Key, fileScored[t][0].Value)

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				p.Logger.Trace().Msgf("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
				continue
			}

			for _, a := range p.Assets {
				if topScored.Key == a.GetName() {
					p.Binary = a
					break
				}
			}
		}

		if p.Binary != nil {
			break
		}
	}

	if p.Binary == nil {
		return errors.New("no binary found")
	}

	return nil
}

func (p *Provider) discoverChecksum() error {
	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	p.Logger.Trace().Msgf("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		p.Logger.Trace().Msgf("discover: type: %d, files: %d", k, len(v))
	}

	// Note: second pass we want to look for everything else, using binary results to help score the remaining assets
	// THis is for the checksum, signature and key files
	for k, v := range fileScoring {
		if k != asset.Checksum {
			continue
		}

		ext := []string{"sha256", "md5", "sha1", "txt", "sha256sum", "sha1sum", "md5sum", "sha512", "sha512sum"}
		if _, ok := fileScored[k]; !ok {
			fileScored[k] = []score.Sorted{}
		}

		fileScored[k] = score.Score(v, &score.Options{
			OS:         p.OSConfig.GetAliases(),
			Arch:       p.OSConfig.GetArchitectures(),
			Names:      []string{p.Binary.GetName()},
			Extensions: ext,
			WeightedTerms: map[string]int{
				"checksums": 100,
				"SHA512":    50,
				"sha512sum": 50,
				"SHA256":    40,
				"sha256sum": 40,
				"MD5":       30,
				"md5sum":    30,
				"SHA1":      20,
				"sha1sum":   20,
				"SHA":       15,
				"shasum":    15,
				"SUMS":      10,
			},
			InvalidOS:   p.OSConfig.InvalidOS(),
			InvalidArch: p.OSConfig.InvalidArchitectures(),
		})

		if len(fileScored[k]) > 0 {
			for _, vv := range fileScored[k] {
				p.Logger.Debug().Msgf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	// Note: we want to look for the best binary by looking at binaries, archives and unknowns
	for _, t := range []asset.Type{asset.Checksum} {
		if len(fileScored[t]) > 0 {
			p.Logger.Trace().
				Str("search_type", "checksum").
				Str("filename", fileScored[t][0].Key).
				Int("score", fileScored[t][0].Value).
				Str("filetype", t.String()).
				Msg("scored")

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				p.Logger.Trace().Msgf("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
				continue
			}
			for _, a := range p.Assets {
				if topScored.Key == a.GetName() {
					p.Checksum = a
					break
				}
			}
		}

		if p.Checksum != nil {
			break
		}
	}

	return nil
}

func (p *Provider) determineChecksumSigTypes() error {
	p.ChecksumType = "none"
	if p.Checksum != nil {
		p.ChecksumType = p.Checksum.GetChecksumType()
	}

	p.SignatureType = SignatureTypeNone
	for _, a := range p.Assets {
		if a.GetType() != asset.Signature {
			continue
		}

		if p.SignatureType == SignatureTypeFile {
			break
		}

		if a.GetParentType() == asset.Binary || a.GetParentType() == asset.Archive || a.GetParentType() == asset.Unknown {
			p.SignatureType = SignatureTypeFile
		} else if a.GetParentType() == asset.Checksum {
			p.SignatureType = SignatureTypeChecksum
		}
	}

	p.Logger.Trace().Msgf("checksum type: %s", p.ChecksumType)
	p.Logger.Trace().Msgf("signature type: %s", p.SignatureType)

	return nil
}

func (p *Provider) discoverSignature(version string) error { //nolint:gocyclo
	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	p.Logger.Trace().Msgf("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		p.Logger.Trace().Msgf("discover: type: %d, files: %d", k, len(v))
	}

	var names []string
	switch p.SignatureType {
	case SignatureTypeChecksum:
		names = append(names, p.Checksum.GetName())
		for _, ext := range []string{"sig", "asc"} {
			names = append(names, fmt.Sprintf("%s.%s", p.Checksum.GetName(), ext))
		}
	case SignatureTypeFile:
		names = append(names, p.Binary.GetName())
		for _, ext := range []string{"sig", "asc"} {
			names = append(names, fmt.Sprintf("%s.%s", p.Binary.GetName(), ext))
		}
	}

	// Note: second pass we want to look for everything else, using binary results to help score the remaining assets
	// This is for the checksum, signature and key files
	for k, v := range fileScoring {
		if k != asset.Signature {
			continue
		}

		ext := []string{"sig", "asc", "sig.asc", "gpg", "keyless.sig"}
		var detectedOS []string
		var arch []string

		if _, ok := fileScored[k]; !ok {
			fileScored[k] = []score.Sorted{}
		}

		p.Logger.Trace().Msgf("names: %v", names)

		fileScored[k] = score.Score(v, &score.Options{
			OS:          detectedOS,
			Arch:        arch,
			Extensions:  ext,
			Names:       names,
			Versions:    []string{version},
			InvalidOS:   p.OSConfig.InvalidOS(),
			InvalidArch: p.OSConfig.InvalidArchitectures(),
		})

		if len(fileScored[k]) > 0 {
			for _, vv := range fileScored[k] {
				p.Logger.Debug().Msgf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	// Note: we want to look for the best binary by looking at binaries, archives and unknowns
	for _, t := range []asset.Type{asset.Signature} {
		if len(fileScored[t]) > 0 {
			p.Logger.Trace().Str("type", "signature").Msgf("top scored (%d): %s (%d)", t, fileScored[t][0].Key, fileScored[t][0].Value)

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				p.Logger.Trace().Str("type", "signature").Msgf("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
				continue
			}
			for _, a := range p.Assets {
				if topScored.Key == a.GetName() {
					p.Signature = a
					p.Key = a.GetMatchedAsset()
					break
				}
			}
		}

		if p.Signature != nil {
			break
		}
	}

	return nil
}

// TODO: refactor into smaller functions for testing
func (p *Provider) discoverMatch() error { //nolint:gocyclo
	// Match keys to signatures.
	for _, a := range p.Assets {
		if a.GetType() != asset.Signature {
			continue
		}

		if a.GetMatchedAsset() != nil {
			continue
		}

		sigBaseName := strings.TrimSuffix(a.GetName(), filepath.Ext(a.GetName()))

		p.Logger.Trace().Msgf("signature base name: %s", sigBaseName)

		for _, aa := range p.Assets {
			if aa.GetType() != asset.Key {
				continue
			}

			keyBaseName := strings.TrimSuffix(aa.GetName(), filepath.Ext(aa.GetName()))

			p.Logger.Trace().Msgf("key base name: %s %s", keyBaseName, aa.GetBaseName())

			if strings.EqualFold(keyBaseName, sigBaseName) || strings.EqualFold(a.GetBaseName(), aa.GetBaseName()) {
				p.Logger.Trace().Msgf("matched key: %s to signature: %s", aa.GetName(), a.GetName())
				a.SetMatchedAsset(aa)
				aa.SetMatchedAsset(a)
				break
			}
		}
	}

	// Match remaining keys to signatures, hopefully there's only a single key remaining
	// TODO: what to do if there are multiple keys remaining? (Maybe support multiple matched???)
	// Use Case: Keyless vs Keyed signing, cosign does both. The keyed file is used for multiple files.
	for _, a := range p.Assets {
		if a.GetType() != asset.Key {
			continue
		}

		if a.GetMatchedAsset() != nil {
			continue
		}

		p.Logger.Trace().Msgf("unmatched key: %s", a.GetName())

		for _, b := range p.Assets {
			if b.GetType() != asset.Signature {
				continue
			}

			if b.GetMatchedAsset() != nil {
				continue
			}

			b.SetMatchedAsset(a)
			p.Logger.Trace().Msgf("matched key: %s to signature: %s", a.GetName(), b.GetName())
		}
	}

	foundGPG := false

	for _, a := range p.Assets {
		if a.GetType() != asset.Signature {
			continue
		}

		if a.GetMatchedAsset() != nil {
			continue
		}

		if !strings.HasSuffix(a.GetName(), ".asc") {
			continue
		}

		keyName := strings.ReplaceAll(a.GetName(), ".asc", ".pub")

		gpgAsset := &GPGAsset{
			Asset:   asset.New(keyName, "", p.GetOS(), p.GetArch(), ""),
			Options: p.GetOptions(),
		}

		p.Logger.Trace().Str("sig", a.GetName()).Str("key", gpgAsset.GetName()).Msg("matched asset")

		if !foundGPG {
			p.Logger.Info().Msg("gpg detected will fetch public key for signature")
			foundGPG = true
		}

		gpgAsset.SetMatchedAsset(a)
		a.SetMatchedAsset(gpgAsset)

		p.Assets = append(p.Assets, gpgAsset)
	}

	return nil
}

// Discover will attempt to discover and categorize the assets provided
func (p *Provider) Discover(names []string, version string) error {
	if err := p.discoverMatch(); err != nil {
		return err
	}

	if err := p.discoverBinary(names, version); err != nil {
		return err
	}

	if err := p.discoverChecksum(); err != nil {
		return err
	}

	if err := p.determineChecksumSigTypes(); err != nil {
		return err
	}

	if err := p.discoverSignature(version); err != nil {
		return err
	}

	return nil
}

func (p *Provider) Download(ctx context.Context) error {
	p.Logger.Info().Msg("downloading assets")

	if p.Binary != nil {
		if err := p.Binary.Download(ctx); err != nil {
			return err
		}
	}

	if p.Signature != nil {
		if err := p.Signature.Download(ctx); err != nil {
			return err
		}
	}

	if p.Checksum != nil {
		if err := p.Checksum.Download(ctx); err != nil {
			return err
		}
	}

	if p.Key != nil {
		if err := p.Key.Download(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (p *Provider) Verify() error {
	if err := p.verifyChecksum(); err != nil {
		return err
	}

	return p.verifySignature()
}

func (p *Provider) verifySignature() error {
	if v, ok := p.Options.Settings["no-signature-verify"]; ok && v.(bool) {
		p.Logger.Warn().Msg("skipping signature verification (user-requested)")
		return nil
	}

	if p.Signature == nil {
		switch p.Options.Config.Settings.SignatureMissing {
		case common.Ignore:
			return nil
		case common.Warn:
			p.Logger.Warn().Msg("skipping signature verification (no signature)")
			return nil
		case common.Error:
			return errors.New("signature verification failed (no signature)")
		}
	}

	// TODO: better pgp detection
	if strings.HasSuffix(p.Signature.GetName(), ".asc") {
		return p.verifyGPGSignature()
	}

	return p.verifyCosignSignature()
}

func (p *Provider) verifyGPGSignature() error {
	var filePath string
	if p.SignatureType == "checksum" {
		filePath = p.Checksum.GetFilePath()
	} else {
		filePath = p.Binary.GetFilePath()
	}

	publicKeyPath := p.Key.GetFilePath()
	signaturePath := p.Signature.GetFilePath()

	publicKeyContent, err := os.Open(publicKeyPath)
	if err != nil {
		return err
	}

	signatureContent, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("failed to read signature file: %w", err)
	}

	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file to be verified: %w", err)
	}

	keyObj, err := crypto.NewKeyFromArmoredReader(publicKeyContent)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	keyRing, err := crypto.NewKeyRing(keyObj)
	if err != nil {
		return fmt.Errorf("failed to create keyring: %w", err)
	}

	message := crypto.NewPlainMessage(fileContent)
	signature, err := crypto.NewPGPSignatureFromArmored(string(signatureContent))
	if err != nil {
		signature = crypto.NewPGPSignature(signatureContent)
	}

	err = keyRing.VerifyDetached(message, signature, crypto.GetUnixTime())
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	p.Logger.Info().Msg("gpg signature verified")
	return nil
}

// TODO: refactor and clean up for the different signature verification methods
func (p *Provider) verifyCosignSignature() error { //nolint:gocyclo
	var bundle *cosign.Bundle
	if p.Key == nil {
		sigData, err := os.ReadFile(p.Signature.GetFilePath())
		if err != nil {
			return err
		}
		if err := json.Unmarshal(sigData, &bundle); err != nil {
			p.Logger.Debug().Err(err).Msg("unable to parse json for bundle signature")
		}

		if bundle == nil {
			p.Logger.Warn().Msg("skipping signature verification (no key)")
			return nil
		}
	}

	p.Logger.Trace().Msg("verifying signature")

	var fileContent []byte
	var err error
	if p.SignatureType == "checksum" {
		p.Logger.Trace().Msgf("verifying checksum signature: %s", p.Checksum.GetName())
		fileContent, err = os.ReadFile(p.Checksum.GetFilePath())
		if err != nil {
			return err
		}
	} else {
		p.Logger.Trace().Msg("verifying binary signature")
		fileContent, err = os.ReadFile(p.Binary.GetFilePath())
		if err != nil {
			return err
		}
	}

	var sigData []byte
	var publicKeyContentEncoded []byte
	if p.Key != nil {
		p.Logger.Trace().Msgf("key file name: %s", p.Key.GetName())
		publicKeyContentEncoded, err = os.ReadFile(p.Key.GetFilePath())
		if err != nil {
			return err
		}

		sigData, err = os.ReadFile(p.Signature.GetFilePath())
		if err != nil {
			return err
		}
	} else if bundle != nil {
		p.Logger.Trace().Msg("key file name via bundle")
		publicKeyContentEncoded = []byte(bundle.Certificate)
		sigData = []byte(bundle.Signature)
	}

	publicKeyContent, err := base64.StdEncoding.DecodeString(string(publicKeyContentEncoded))
	if err != nil {
		if errors.Is(err, base64.CorruptInputError(0)) {
			publicKeyContent = publicKeyContentEncoded
		} else {
			return err
		}
	}

	pubKey, err := cosign.ParsePublicKey(publicKeyContent)
	if err != nil {
		return err
	}

	p.Logger.Trace().Msgf("signature file name: %s", p.Signature.GetName())

	dataHash := cosign.HashData(fileContent)

	valid, err := cosign.VerifySignature(pubKey, dataHash, sigData)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("unable to validate signature")
	}

	p.Logger.Info().Msg("signature verified")
	return nil
}

// verifyChecksum - verify the checksum of the binary
func (p *Provider) verifyChecksum() error {
	if v, ok := p.Options.Settings["no-checksum-verify"]; ok && v.(bool) {
		p.Logger.Warn().Msg("skipping checksum verification (user-requested)")
		return nil
	}

	if p.Checksum == nil {
		switch p.Options.Config.Settings.ChecksumMissing {
		case common.Ignore:
			return nil
		case common.Warn:
			p.Logger.Warn().Msg("skipping checksum verification (no checksum)")
			return nil
		case common.Error:
			return errors.New("checksum verification failed (no checksum)")
		}
	}

	p.Logger.Debug().Msg("verifying checksum")
	p.Logger.Trace().Msgf("binary: %s", p.Binary.GetName())

	match, err := checksum.CompareHashWithChecksumFile(p.Binary.GetName(),
		p.Binary.GetFilePath(), p.Checksum.GetFilePath())
	if err != nil {
		if errors.Is(err, checksum.ErrUnsupportedHashLength) {
			switch p.Options.Config.Settings.ChecksumUnknown {
			case common.Warn:
				p.Logger.Warn().Msg("skipping checksum verification (unsupported hash length)")
			case common.Error:
				return err
			default:
			}

			return nil
		} else {
			return err
		}
	}

	p.Logger.Trace().Msgf("checksum match: %v", match)

	if !match {
		return fmt.Errorf("checksum verification failed")
	}

	p.Logger.Info().Msg("checksum verified")

	return nil
}

func (p *Provider) Extract() error {
	return p.Binary.Extract()
}

func (p *Provider) Install() error {
	return p.Binary.Install(
		p.Binary.ID(), p.Options.Config.BinPath, filepath.Join(p.Options.Config.GetOptPath(), p.Binary.Path()))
}

func (p *Provider) Cleanup() error {
	return p.Binary.Cleanup()
}
