package install

import (
	"fmt"
	"strings"

	"github.com/ekristen/distillery/pkg/osconfig"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/source"
)

func NewSource(src string, opts *provider.Options) (provider.ISource, error) { //nolint:funlen,gocyclo
	detectedOS := osconfig.New(opts.OS, opts.Arch)

	// Extract binary hint (e.g. owner/repo:logcli@version or owner/repo@version:logcli)
	src, hint := extractHint(src)
	if hint != "" {
		if opts.Settings == nil {
			opts.Settings = map[string]interface{}{}
		}
		opts.Settings["binary-hint"] = hint
	}

	version := "latest"
	versionParts := strings.Split(src, "@")
	if len(versionParts) > 1 {
		src = versionParts[0]
		version = versionParts[1]
	}

	parts := strings.Split(src, "/")

	providerConfig := provider.Provider{Options: opts, OSConfig: detectedOS, Logger: opts.Logger}

	if len(parts) == 1 {
		switch opts.Config.DefaultSource {
		case source.HomebrewSource:
			return &source.Homebrew{
				Provider: providerConfig,
				Formula:  parts[0],
				Version:  version,
			}, nil
		case source.HashicorpSource:
			return &source.Hashicorp{
				Provider: providerConfig,
				Owner:    parts[0],
				Repo:     parts[0],
				Version:  version,
			}, nil
		case source.KubernetesSource:
			return &source.Kubernetes{
				GitHub: source.GitHub{
					Provider: providerConfig,
					Owner:    source.KubernetesSource,
					Repo:     source.KubernetesSource,
					Version:  version,
				},
				AppName: parts[0],
			}, nil
		}

		return nil, fmt.Errorf("invalid install source, expect alias or format of owner/repo or owner/repo@version")
	}

	if len(parts) == 2 {
		// could be GitHub or Homebrew or Hashicorp
		if parts[0] == source.HomebrewSource { //nolint:staticcheck
			return &source.Homebrew{
				Provider: providerConfig,
				Formula:  parts[1],
				Version:  version,
			}, nil
		} else if parts[0] == source.HashicorpSource { //nolint:dupl
			return &source.Hashicorp{
				Provider: providerConfig,
				Owner:    parts[1],
				Repo:     parts[1],
				Version:  version,
			}, nil
		} else if parts[0] == source.KubernetesSource {
			return &source.Kubernetes{
				GitHub: source.GitHub{
					Provider: providerConfig,
					Owner:    source.KubernetesSource,
					Repo:     source.KubernetesSource,
					Version:  version,
				},
				AppName: parts[1],
			}, nil
		} else if parts[0] == source.HelmSource {
			return &source.Helm{
				GitHub: source.GitHub{
					Provider: providerConfig,
					Owner:    source.HelmSource,
					Repo:     source.HelmSource,
					Version:  version,
				},
				AppName: parts[1],
			}, nil
		}

		switch opts.Config.DefaultSource {
		case source.GitHubSource:
			return &source.GitHub{
				Provider: providerConfig,
				Owner:    parts[0],
				Repo:     parts[1],
				Version:  version,
			}, nil
		case source.GitLabSource:
			owner := strings.Join(parts[1:len(parts)-1], "/")
			repo := parts[len(parts)-1]

			return &source.GitLab{
				Provider: providerConfig,
				Owner:    owner,
				Repo:     repo,
				Version:  version,
			}, nil
		}

		return nil, fmt.Errorf("invalid install source, expect alias	 or format of owner/repo or owner/repo@version")
	} else if len(parts) >= 3 {
		if strings.HasPrefix(parts[0], source.GitHubSource) {
			if parts[1] == source.HashicorpSource { //nolint:dupl,staticcheck
				return &source.Hashicorp{
					Provider: providerConfig,
					Owner:    parts[1],
					Repo:     parts[2],
					Version:  version,
				}, nil
			} else if parts[1] == source.KubernetesSource {
				return &source.Kubernetes{
					GitHub: source.GitHub{
						Provider: providerConfig,
						Owner:    source.KubernetesSource,
						Repo:     source.KubernetesSource,
						Version:  version,
					},
					AppName: parts[2],
				}, nil
			} else if parts[1] == source.HelmSource {
				return &source.Helm{
					GitHub: source.GitHub{
						Provider: providerConfig,
						Owner:    source.HelmSource,
						Repo:     source.HelmSource,
						Version:  version,
					},
					AppName: parts[2],
				}, nil
			}

			return &source.GitHub{
				Provider: providerConfig,
				Owner:    parts[1],
				Repo:     parts[2],
				Version:  version,
			}, nil
		} else if strings.HasPrefix(parts[0], source.GitLabSource) {
			owner := strings.Join(parts[1:len(parts)-1], "/")
			repo := parts[len(parts)-1]

			return &source.GitLab{
				Provider: providerConfig,
				Owner:    owner,
				Repo:     repo,
				Version:  version,
			}, nil
		} else if strings.HasPrefix(parts[0], source.CodebergSource) {
			owner := parts[1]
			repo := strings.Join(parts[2:], "/")

			return &source.Forgejo{
				Provider:   providerConfig,
				BaseURL:    source.CodebergBaseURL,
				SourceName: source.CodebergSource,
				Owner:      owner,
				Repo:       repo,
				Version:    version,
			}, nil
		}

		for pn, p := range opts.Config.Providers {
			if pn == parts[0] {
				switch p.Provider {
				case source.GitLabSource:
					owner := strings.Join(parts[1:len(parts)-1], "/")
					repo := parts[len(parts)-1]

					return &source.GitLab{
						Provider: providerConfig,
						BaseURL:  p.BaseURL,
						Owner:    owner,
						Repo:     repo,
						Version:  version,
					}, nil
				case source.ForgejoSource:
					owner := parts[1]
					repo := strings.Join(parts[2:], "/")

					return &source.Forgejo{
						Provider:   providerConfig,
						BaseURL:    p.BaseURL,
						SourceName: pn,
						Owner:      owner,
						Repo:       repo,
						Version:    version,
					}, nil
				}
			}
		}

		return nil, fmt.Errorf("unknown source: %s", src)
	}

	return nil, fmt.Errorf("unknown source: %s", src)
}

// extractHint extracts a binary hint from a source string.
// Supports both owner/repo:hint@version and owner/repo@version:hint formats.
// Returns the source string with the hint removed, and the hint itself.
func extractHint(src string) (cleaned, hint string) {
	idx := strings.Index(src, ":")
	if idx == -1 {
		return src, ""
	}

	hint = src[idx+1:]
	cleaned = src[:idx]

	// If hint contains @, the version was after the hint (e.g. repo:hint@version)
	if atIdx := strings.Index(hint, "@"); atIdx != -1 {
		version := hint[atIdx+1:]
		hint = hint[:atIdx]
		cleaned = cleaned + "@" + version
	}

	return cleaned, hint
}
