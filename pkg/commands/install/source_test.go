package install

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/provider"
	"github.com/ekristen/distillery/pkg/source"
)

func Test_NewSource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		source        string
		defaultSource string
		error         bool
		want          provider.ISource
		wantBaseURL   string
		setupConfig   func(*config.Config)
	}{
		{
			source: "ekristen/aws-nuke",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "latest",
			},
		},
		{
			source: "github/ekristen/aws-nuke",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "latest",
			},
		},
		{
			source: "github.com/ekristen/aws-nuke",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "latest",
			},
		},
		{
			source: "ekristen/aws-nuke@3.1.1",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "3.1.1",
			},
		},
		{
			source: "github/ekristen/aws-nuke@3.1.1",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "3.1.1",
			},
		},
		{
			source: "github.com/ekristen/aws-nuke@3.1.1",
			want: &source.GitHub{
				Owner:   "ekristen",
				Repo:    "aws-nuke",
				Version: "3.1.1",
			},
		},
		{
			source: "homebrew/aws-nuke",
			want: &source.Homebrew{
				Formula: "aws-nuke",
				Version: "latest",
			},
		},
		{
			source: "hashicorp/terraform",
			want: &source.Hashicorp{
				Owner: "hashicorp",
				Repo:  "terraform",
			},
		},
		{
			source:        "opentufu",
			defaultSource: "homebrew",
			want: &source.Homebrew{
				Formula: "opentufu",
				Version: "latest",
			},
		},
		{
			source:        "terraform",
			defaultSource: "hashicorp",
			want: &source.Hashicorp{
				Owner: "hashicorp",
				Repo:  "terraform",
			},
		},
		{
			source:        "gitlab-org/gitlab-runner",
			defaultSource: "gitlab",
			want: &source.GitLab{
				Owner: "gitlab-org",
				Repo:  "gitlab-runner",
			},
		},
		{
			source:        "terraform",
			defaultSource: "unknown",
			error:         true,
			want:          nil,
		},
		{
			source: "github/hashicorp/terraform",
			want: &source.Hashicorp{
				Owner: "hashicorp",
				Repo:  "terraform",
			},
		},
		{
			source: "gitlab/gitlab-org/gitlab-runner",
			want: &source.GitLab{
				Owner: "gitlab-org",
				Repo:  "gitlab-runner",
			},
		},
		{
			source:        "unknown/unknown",
			defaultSource: "unknown",
			error:         true,
		},
		{
			source: "unknown/some-owner/some-repo",
			error:  true,
		},
		{
			source: "unknown/some-owner/some-repo/extra@3.1.1",
			error:  true,
		},
		// Codeberg shorthand
		{
			source: "codeberg/owner/repo",
			want: &source.Forgejo{
				Owner:      "owner",
				Repo:       "repo",
				Version:    "latest",
				SourceName: source.CodebergSource,
			},
			wantBaseURL: source.CodebergBaseURL,
		},
		{
			source: "codeberg/owner/repo@2.0.0",
			want: &source.Forgejo{
				Owner:      "owner",
				Repo:       "repo",
				Version:    "2.0.0",
				SourceName: source.CodebergSource,
			},
			wantBaseURL: source.CodebergBaseURL,
		},
		// forgejo as defaultSource is not supported — requires an explicit base URL via a
		// configured provider, so owner/repo with no prefix should error.
		{
			source:        "owner/repo",
			defaultSource: "forgejo",
			error:         true,
		},
		// bare forgejo/owner/repo without a matching config provider is not supported
		{
			source: "forgejo/owner/repo",
			error:  true,
		},
		// Config-defined Forgejo provider
		{
			source: "myforgejo/someowner/somerepo",
			want: &source.Forgejo{
				Owner:      "someowner",
				Repo:       "somerepo",
				Version:    "latest",
				SourceName: "myforgejo",
			},
			wantBaseURL: "https://git.example.com/api/v1",
			setupConfig: func(cfg *config.Config) {
				cfg.Providers = map[string]*config.Provider{
					"myforgejo": {
						Provider: "forgejo",
						BaseURL:  "https://git.example.com/api/v1",
					},
				}
			},
		},
		{
			source: "myforgejo/someowner/somerepo@1.2.3",
			want: &source.Forgejo{
				Owner:      "someowner",
				Repo:       "somerepo",
				Version:    "1.2.3",
				SourceName: "myforgejo",
			},
			wantBaseURL: "https://git.example.com/api/v1",
			setupConfig: func(cfg *config.Config) {
				cfg.Providers = map[string]*config.Provider{
					"myforgejo": {
						Provider: "forgejo",
						BaseURL:  "https://git.example.com/api/v1",
					},
				}
			},
		},
	}

	for _, tt := range cases {
		t.Run(tt.source, func(t *testing.T) {
			cfg, err := config.New("/tmp/test/path")
			assert.NoError(t, err)

			if tt.defaultSource != "" {
				cfg.DefaultSource = tt.defaultSource
			}

			if tt.setupConfig != nil {
				tt.setupConfig(cfg)
			}

			got, err := NewSource(tt.source, &provider.Options{Config: cfg})
			if tt.error {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, tt.want.GetSource(), got.GetSource())

			if tt.wantBaseURL != "" {
				fg, ok := got.(*source.Forgejo)
				assert.True(t, ok, "expected *source.Forgejo, got %T", got)
				if ok {
					assert.Equal(t, tt.wantBaseURL, fg.BaseURL)
					assert.Equal(t, tt.want.(*source.Forgejo).Version, fg.Version)
				}
			}
		})
	}
}
