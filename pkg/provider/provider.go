package provider

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/sirupsen/logrus"

	"github.com/ProtonMail/gopenpgp/v2/crypto"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/checksum"
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

// CommonRun - common run logic for all sources that includes download, extract, install and cleanup
func (p *Provider) CommonRun(ctx context.Context) error {
	if err := p.Download(ctx); err != nil {
		return err
	}

	defer func(s *Provider) {
		err := s.Cleanup()
		if err != nil {
			log.WithError(err).Error("unable to cleanup")
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
	logger := logrus.WithField("discover", "binary")
	logger.Tracef("names: %v", names)

	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	logger.Tracef("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		logger.Tracef("discover: type: %d, files: %d", k, len(v))
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

		fileScored[k] = score.Score(v, &score.Options{
			OS:         detectedOS,
			Arch:       arch,
			Extensions: ext,
			Terms:      names,
			WeightedTerms: map[string]int{
				"source": -20, // as in source.tar.gz
			},
			Versions:          []string{version},
			InvalidOS:         p.OSConfig.InvalidOS(),
			InvalidArch:       p.OSConfig.InvalidArchitectures(),
			InvalidExtensions: []string{".zst"},
		})

		if len(fileScored[k]) > 0 {
			for _, vv := range fileScored[k] {
				if vv.Value >= 40 {
					highEnoughScore = true
				}
				logger.Debugf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	if !highEnoughScore && !p.Options.Settings["no-score-check"].(bool) {
		log.Error("no matching asset found, score too low")
		for _, t := range []asset.Type{asset.Binary, asset.Unknown, asset.Archive} {
			for _, v := range fileScored[t] {
				if v.Value < 40 {
					log.Errorf("closest matching: %s (%d) (threshold: 40) -- override with --no-score-check", v.Key, v.Value)
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
			logger.Tracef("top scored (%d): %s (%d)", t, fileScored[t][0].Key, fileScored[t][0].Value)

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				logger.Tracef("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
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
	logger := logrus.WithField("discover", "checksum")

	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	logger.Tracef("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		logger.Tracef("discover: type: %d, files: %d", k, len(v))
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
				"checksums": 80,
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
				logger.Debugf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	// Note: we want to look for the best binary by looking at binaries, archives and unknowns
	for _, t := range []asset.Type{asset.Checksum} {
		if len(fileScored[t]) > 0 {
			logger.Tracef("top scored (%d): %s (%d)", t, fileScored[t][0].Key, fileScored[t][0].Value)

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				logger.Tracef("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
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
	logger := logrus.WithField("discover", "check-sig-type")

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

	logger.Tracef("checksum type: %s", p.ChecksumType)
	logger.Tracef("signature type: %s", p.SignatureType)

	return nil
}

func (p *Provider) discoverSignature(version string) error { //nolint:gocyclo
	logger := logrus.WithField("discover", "signature")

	fileScoring := map[asset.Type][]string{}
	fileScored := map[asset.Type][]score.Sorted{}

	logger.Tracef("discover: starting - %d", len(p.Assets))

	for _, a := range p.Assets {
		if _, ok := fileScoring[a.GetType()]; !ok {
			fileScoring[a.GetType()] = []string{}
		}
		fileScoring[a.GetType()] = append(fileScoring[a.GetType()], a.GetName())
	}

	for k, v := range fileScoring {
		logger.Tracef("discover: type: %d, files: %d", k, len(v))
	}

	var names []string
	if p.SignatureType == SignatureTypeChecksum {
		names = append(names, p.Checksum.GetName())
		for _, ext := range []string{"sig", "asc"} {
			names = append(names, fmt.Sprintf("%s.%s", p.Checksum.GetName(), ext))
		}
	} else if p.SignatureType == SignatureTypeFile {
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

		logger.Tracef("names: %v", names)

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
				logger.Debugf("file scoring sorted ! type: %d, scored: %v", k, vv)
			}
		}
	}

	// Note: we want to look for the best binary by looking at binaries, archives and unknowns
	for _, t := range []asset.Type{asset.Signature} {
		if len(fileScored[t]) > 0 {
			logger.WithField("type", "signature").Tracef("top scored (%d): %s (%d)", t, fileScored[t][0].Key, fileScored[t][0].Value)

			topScored := fileScored[t][0]
			if topScored.Value < 40 {
				logger.WithField("type", "signature").Tracef("skipped > (%d) too low: %s (%d)", t, topScored.Key, topScored.Value)
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
	logger := logrus.WithField("discover", "match")

	// Match keys to signatures.
	for _, a := range p.Assets {
		if a.GetType() != asset.Signature {
			continue
		}

		if a.GetMatchedAsset() != nil {
			continue
		}

		for _, aa := range p.Assets {
			if aa.GetType() != asset.Key {
				continue
			}

			childS := strings.TrimSuffix(aa.GetName(), filepath.Ext(aa.GetName()))
			parentS := strings.TrimSuffix(a.GetName(), filepath.Ext(a.GetName()))

			if strings.EqualFold(childS, parentS) {
				logger.Tracef("matched key: %s to signature: %s", aa.GetName(), a.GetName())
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

		logger.Tracef("unmatched key: %s", a.GetName())

		for _, b := range p.Assets {
			if b.GetType() != asset.Signature {
				continue
			}

			if b.GetMatchedAsset() != nil {
				continue
			}

			b.SetMatchedAsset(a)
			logger.Tracef("matched key: %s to signature: %s", a.GetName(), b.GetName())
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
			Asset: asset.New(keyName, "", p.GetOS(), p.GetArch(), ""),
		}

		logrus.WithField("sig", a.GetName()).WithField("key", gpgAsset.GetName()).Trace("matched asset")

		if !foundGPG {
			log.Info("gpg detected will fetch public key for signature")
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
	log.Info("downloading assets")
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
		log.Warn("skipping signature verification (user-requested)")
		return nil
	}

	if p.Signature == nil {
		if p.Options.Config.Settings.SignatureMissing == "ignore" {
			return nil
		} else if p.Options.Config.Settings.SignatureMissing == "warn" {
			log.Warn("skipping signature verification (no signature)")
			return nil
		} else if p.Options.Config.Settings.SignatureMissing == "error" {
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

	log.Info("signature verified")

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
			log.WithError(err).Debug("unable to parse json for bundle signature")
		}

		if bundle == nil {
			log.Warn("skipping signature verification (no key)")
			return nil
		}
	}

	logrus.Trace("verifying signature")

	var fileContent []byte
	var err error
	if p.SignatureType == "checksum" {
		logrus.Trace("verifying checksum signature", p.Checksum.GetName())
		fileContent, err = os.ReadFile(p.Checksum.GetFilePath())
		if err != nil {
			return err
		}
	} else {
		logrus.Trace("verifying binary signature")
		fileContent, err = os.ReadFile(p.Binary.GetFilePath())
		if err != nil {
			return err
		}
	}

	var sigData []byte
	var publicKeyContentEncoded []byte
	if p.Key != nil {
		logrus.Trace("key file name: ", p.Key.GetName())
		publicKeyContentEncoded, err = os.ReadFile(p.Key.GetFilePath())
		if err != nil {
			return err
		}

		sigData, err = os.ReadFile(p.Signature.GetFilePath())
		if err != nil {
			return err
		}
	} else if bundle != nil {
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

	logrus.Trace("signature file name: ", p.Signature.GetName())

	dataHash := cosign.HashData(fileContent)

	valid, err := cosign.VerifySignature(pubKey, dataHash, sigData)
	if err != nil {
		return err
	}

	if !valid {
		return errors.New("unable to validate signature")
	}

	log.Info("signature verified")

	return nil
}

// verifyChecksum - verify the checksum of the binary
func (p *Provider) verifyChecksum() error {
	if v, ok := p.Options.Settings["no-checksum-verify"]; ok && v.(bool) {
		log.Warn("skipping checksum verification (user-requested)")
		return nil
	}

	if p.Checksum == nil {
		if p.Options.Config.Settings.ChecksumMissing == "ignore" {
			return nil
		} else if p.Options.Config.Settings.ChecksumMissing == "warn" {
			log.Warn("skipping checksum verification (no checksum)")
			return nil
		} else if p.Options.Config.Settings.ChecksumMissing == "error" {
			return errors.New("checksum verification failed (no checksum)")
		}
	}

	logrus.Debug("verifying checksum")
	logrus.Tracef("binary: %s", p.Binary.GetName())

	// TODO: support other checksum functions
	match, err := checksum.CompareHashWithChecksumFile(p.Binary.GetName(),
		p.Binary.GetFilePath(), p.Checksum.GetFilePath(), sha256.New)
	if err != nil {
		return err
	}

	logrus.Tracef("checksum match: %v", match)

	if !match {
		return fmt.Errorf("checksum verification failed")
	}

	log.Info("checksum verified")

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
