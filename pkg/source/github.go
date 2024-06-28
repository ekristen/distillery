package source

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/google/go-github/v62/github"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/sirupsen/logrus"

	"github.com/ekristen/distillery/pkg/common"
)

type GitHub struct {
	Source

	Owner   string
	Repo    string
	Version string

	GoReleaser *GoReleaser

	Release      *github.RepositoryRelease
	Assets       []*github.ReleaseAsset
	MatchedAsset *github.ReleaseAsset

	client *github.Client
}

func (s *GitHub) GetSource() string {
	return "github"
}
func (s *GitHub) GetOwner() string {
	return s.Owner
}
func (s *GitHub) GetRepo() string {
	return s.Repo
}
func (s *GitHub) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}

func (s *GitHub) GetID() string {
	return strings.Join([]string{s.GetSource(), s.GetOwner(), s.GetRepo(), s.GetOS(), s.GetArch()}, "-")
}

func (s *GitHub) Run(ctx context.Context, version, githubToken string) error {
	cacheFile := filepath.Join(s.Options.MetadataDir, fmt.Sprintf("cache-%s", s.GetID()))

	s.client = github.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())
	if githubToken != "" {
		s.client = s.client.WithAuthToken(githubToken)
	}

	if err := s.FindRelease(ctx); err != nil {
		return err
	}

	if err := s.GetReleaseAssets(ctx); err != nil {
		return err
	}

	s.DetectGoReleaser()

	if err := s.FindReleaseAsset(); err != nil {
		return err
	}

	if err := s.Download(ctx, version, githubToken); err != nil {
		return err
	}

	if err := s.ExtractInstall(s.GetRepo(), s.GetOS(), s.GetArch(), s.Version); err != nil {
		return err
	}

	return nil
}

// FindRelease - query API to find the version being sought or return an error
func (s *GitHub) FindRelease(ctx context.Context) error {
	var err error
	var release *github.RepositoryRelease

	if s.Version == "latest" {
		release, _, err = s.client.Repositories.GetLatestRelease(ctx, s.GetOwner(), s.GetRepo())
		if err != nil {
			return err
		}
	} else {
		releases, _, err := s.client.Repositories.ListReleases(ctx, s.GetOwner(), s.GetRepo(), nil)
		if err != nil {
			return err
		}
		for _, r := range releases {
			if r.GetTagName() == s.Version || r.GetName() == fmt.Sprintf("v%s", s.Version) {
				release = r
				break
			}
		}
	}

	if release == nil {
		return fmt.Errorf("release not found")
	}

	s.Release = release

	return nil
}

func (s *GitHub) GetReleaseAssets(ctx context.Context) error {
	assets, _, err := s.client.Repositories.ListReleaseAssets(
		ctx, s.GetOwner(), s.GetRepo(), s.Release.GetID(), &github.ListOptions{
			PerPage: 100,
		})
	if err != nil {
		return err
	}

	// TODO: add pagination support

	s.Assets = assets

	return nil
}

func (s *GitHub) DetectGoReleaser() {
	hasChecksum := false
	hasSig := false

	gr := &GoReleaser{}

	for _, asset := range s.Assets {
		logrus.Debugf("asset: %s", asset.GetName())

		if asset.GetName() == "checksum.txt" {
			hasChecksum = true
			gr.ChecksumFile = asset.GetName()
		}

		if strings.HasSuffix(asset.GetName(), ".sig") {
			hasSig = true
			gr.SignatureFile = asset.GetName()
		}

		if strings.HasSuffix(asset.GetName(), ".pem") {
			gr.KeyFile = asset.GetName()
		}
	}

	if hasChecksum && hasSig {
		s.GoReleaser = gr
	}
}

const darwin = "darwin"

// FindReleaseAsset - find the asset that matches the current OS and Arch, if multiple matches are found it
// will attempt to find the best match based on the suffix for the appropriate OS. If no match is found an error
// is returned.
func (s *GitHub) FindReleaseAsset() error { //nolint:funlen,gocyclo
	suffixes := []string{"none"}
	if s.GetOS() == darwin || s.GetOS() == "linux" {
		suffixes = append(suffixes, ".tar.gz", ".zip")
	} else if s.GetOS() == "windows" {
		suffixes = append(suffixes, ".zip", ".exe")
	}

	oses := []string{s.GetOS()}
	if s.GetOS() == darwin {
		oses = append(oses, "macos")
	}

	archs := []string{s.GetOS()}
	if s.GetOS() == darwin {
		archs = append(archs, "universal")
	}

	var matchingAssets = make(map[string][]*github.ReleaseAsset)

	for _, asset := range s.Assets {
		logrus.Debugf("all > asset: %s", asset.GetName())

		name := strings.ToLower(asset.GetName())

		for _, os1 := range oses {
			if strings.Contains(name, os1) {
				matchingAssets["os"] = append(matchingAssets["os"], asset)
			}
		}
	}

	for _, asset := range matchingAssets["os"] {
		logrus.Debugf("os > asset: %s", asset.GetName())

		name := strings.ToLower(asset.GetName())
		for _, arch := range archs {
			if strings.Contains(name, arch) {
				matchingAssets["arch"] = append(matchingAssets["arch"], asset)
			}
		}
	}

	if len(matchingAssets["arch"]) == 1 {
		s.MatchedAsset = matchingAssets["arch"][0]
		return nil
	}

	for _, asset := range matchingAssets["arch"] {
		logrus.Debugf("arch > asset: %s", asset.GetName())

		name := strings.ToLower(asset.GetName())

		for _, suffix := range suffixes {
			if strings.HasSuffix(name, suffix) {
				matchingAssets["suffix"] = append(matchingAssets["suffix"], asset)
			}
		}
	}

	if len(matchingAssets["suffix"]) == 1 {
		s.MatchedAsset = matchingAssets["suffix"][0]
		return nil
	}

	// if we still have multiple matches, pick the tar.gz variant
	for _, asset := range matchingAssets["suffix"] {
		logrus.Debugf("suffix > asset: %s", asset.GetName())

		name := strings.ToLower(asset.GetName())

		if strings.HasSuffix(name, ".tar.gz") {
			s.MatchedAsset = asset
			return nil
		}
	}

	if s.MatchedAsset == nil {
		for _, asset := range matchingAssets["arch"] {
			logrus.Debugf("arch 2 > asset: %s", asset.GetName())

			name := strings.ToLower(asset.GetName())

			ext := filepath.Ext(name)
			if ext == "" {
				s.MatchedAsset = asset
				return nil
			}
		}
	}

	return fmt.Errorf("no matching asset found")
}

func (s *GitHub) Download(ctx context.Context, version, githubToken string) error {
	spin := spinner.New(spinner.CharSets[11], 100*time.Millisecond) // Build our new spinner
	spin.Suffix = " Downloading asset(s)"                           // Set the prefix text
	spin.Start()                                                    // Start the spinner

	rc, url, err := s.client.Repositories.DownloadReleaseAsset(
		ctx, s.GetOwner(), s.GetRepo(), s.MatchedAsset.GetID(), http.DefaultClient)
	if err != nil {
		return err
	}
	defer rc.Close()

	if url != "" {
		logrus.Tracef("url: %s", url)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	downloadsDir := filepath.Join(cacheDir, common.NAME, "downloads")

	filename := fmt.Sprintf("%s-%s-%s-%s", s.GetOwner(), s.GetRepo(), version, s.MatchedAsset.GetName())

	assetFile := filepath.Join(downloadsDir, filename)
	assetFileHash := assetFile + ".sha256"

	stats, err := os.Stat(assetFileHash)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	s.File = assetFile

	if stats != nil {
		logrus.Debugf("file already downloaded: %s", assetFile)
		return nil
	}

	// TODO: verify hash, add overwrite for force.

	hasher := sha256.New()

	// Create a temporary file
	tmpfile, err := os.Create(assetFile)
	if err != nil {
		return err
	}
	defer tmpfile.Close()

	multiWriter := io.MultiWriter(tmpfile, hasher)

	// Write the asset's content to the temporary file
	_, err = io.Copy(multiWriter, rc)
	if err != nil {
		return err
	}

	spin.FinalMSG = "Downloaded complete\n"
	logrus.Tracef("hash: %s", fmt.Sprintf("%x", hasher.Sum(nil)))

	_ = os.WriteFile(assetFile+".sha256", []byte(fmt.Sprintf("%x", hasher.Sum(nil))), 0600)

	logrus.Tracef("Downloaded asset to: %s", tmpfile.Name())

	spin.Stop()

	return nil
}
