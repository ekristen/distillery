package score

import (
	"testing"

	"github.com/h2non/filetype/matchers"
	"github.com/h2non/filetype/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
	logrus.SetReportCaller(true)
}

func TestScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		names    []string
		terms    []string
		opts     *Options
		expected []Sorted
	}{
		{
			name:  "unsupported extension",
			names: []string{"dist-linux-amd64.deb"},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{"unknown"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64.deb",
					Value: 69,
				},
			},
		},
		{
			name: "simple binary",
			names: []string{
				"dist-linux-amd64",
			},
			opts: &Options{
				OS:   []string{"linux"},
				Arch: []string{"amd64"},
				Extensions: []string{
					matchers.TypeGz.Extension,
					types.Unknown.Extension,
					matchers.TypeZip.Extension,
					matchers.TypeXz.Extension,
					matchers.TypeTar.Extension,
					matchers.TypeBz2.Extension,
					matchers.TypeExe.Extension,
				},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64",
					Value: 69,
				},
			},
		},
		{
			name: "unknown binary",
			names: []string{
				"something-linux",
			},
			opts: &Options{
				OS:   []string{"macos"},
				Arch: []string{"amd64"},
				Extensions: []string{
					types.Unknown.Extension,
				},
				Terms: []string{"something"},
			},
			expected: []Sorted{
				{
					Key:   "something-linux",
					Value: 7,
				},
			},
		},
		{
			name: "simple binary matching signature file",
			names: []string{
				"dist-linux-amd64.sig",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{"sig"},
				Terms:      []string{"dist"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64.sig",
					Value: 106,
				},
			},
		},
		{
			name: "simple binary matching key file",
			names: []string{
				"dist-linux-amd64.pem",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{"pem", "pub"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64.pem",
					Value: 109,
				},
			},
		},
		{
			name: "global checksums file",
			names: []string{
				"checksums.txt",
				"SHA256SUMS",
				"SHASUMS",
			},
			opts: &Options{
				OS:         []string{},
				Arch:       []string{},
				Extensions: []string{"txt"},
				Terms: []string{
					"checksums",
				},
			},
			expected: []Sorted{
				{
					Key:   "checksums.txt",
					Value: 40,
				},
				{
					Key:   "SHA256SUMS",
					Value: 10,
				},
				{
					Key:   "SHASUMS",
					Value: 10,
				},
			},
		},
		{
			name: "invalid-os-and-arch",
			names: []string{
				"dist-linux-amd64",
				"dist-windows-arm64.exe",
				"dist-darwin-amd64",
			},
			opts: &Options{
				OS:         []string{"windows"},
				Arch:       []string{"arm64"},
				Extensions: []string{"exe"},
				Terms: []string{
					"dist",
				},
				InvalidOS:   []string{"linux", "darwin"},
				InvalidArch: []string{"amd64"},
			},
			expected: []Sorted{
				{
					Key:   "dist-windows-arm64.exe",
					Value: 106, // os, arch, ext, name match
				},
				{
					Key:   "dist-linux-amd64",
					Value: -68, // invalid os and arch
				},
				{
					Key:   "dist-darwin-amd64",
					Value: -68, // invalid os and arch
				},
			},
		},
		{
			name: "invalid-extensions",
			names: []string{
				"dist-linux-amd64",
				"dist-windows-amd64.exe",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{""},
				Terms: []string{
					"dist",
				},
				InvalidOS:         []string{"windows"},
				InvalidExtensions: []string{"exe"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64",
					Value: 86, // os, arch, name match
				},
				{
					Key:   "dist-windows-amd64.exe",
					Value: -21, // invalid extension and os
				},
			},
		},
		{
			name: "better-match",
			names: []string{
				"nerdctl-1.7.7-linux-arm64.tar.gz",
				"nerdctl-1.7.7-linux-amd64.tar.gz",
				"nerdctl-full-1.7.7-linux-amd64.tar.gz",
				"nerdctl-full-1.7.7-linux-arm64.tar.gz",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Versions:   []string{"1.7.7"},
				Extensions: []string{""},
				Terms: []string{
					"nerdctl",
				},
				InvalidOS:         []string{"windows"},
				InvalidExtensions: []string{"exe"},
			},
			expected: []Sorted{
				{
					Key:   "nerdctl-1.7.7-linux-amd64.tar.gz",
					Value: 88,
				},
				{
					Key:   "nerdctl-full-1.7.7-linux-amd64.tar.gz",
					Value: 83,
				},
				{
					Key:   "nerdctl-1.7.7-linux-arm64.tar.gz",
					Value: 51,
				},
				{
					Key:   "nerdctl-full-1.7.7-linux-arm64.tar.gz",
					Value: 46,
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := Score(c.names, c.opts)
			assert.ElementsMatch(t, c.expected, actual)
		})
	}
}
