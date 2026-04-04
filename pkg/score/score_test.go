package score

import (
	"testing"

	"github.com/h2non/filetype/matchers"
	"github.com/h2non/filetype/types"
	"github.com/stretchr/testify/assert"
)

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
					Value: 89, // os(40) + arch(30) + ext(20) + accuracy(-1)
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
			name: "invalid-terms-penalty",
			names: []string{
				"dist-linux-amd64-musl",
				"dist-linux-amd64",
			},
			opts: &Options{
				OS:           []string{"linux"},
				Arch:         []string{"amd64"},
				Extensions:   []string{""},
				InvalidTerms: []string{"musl"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64",
					Value: 69, // os(40) + arch(30) + accuracy(-1)
				},
				{
					Key:   "dist-linux-amd64-musl",
					Value: 54, // os(40) + arch(30) + invalidterm(-10) + accuracy(-6)
				},
			},
		},
		{
			name: "invalid-library-penalty",
			names: []string{
				"dist-linux-amd64-musl",
				"dist-linux-amd64",
			},
			opts: &Options{
				OS:             []string{"linux"},
				Arch:           []string{"amd64"},
				Extensions:     []string{""},
				InvalidLibrary: []string{"musl"},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64",
					Value: 69, // os(40) + arch(30) + accuracy(-1)
				},
				{
					Key:   "dist-linux-amd64-musl",
					Value: 34, // os(40) + arch(30) + invalidlib(-30) + accuracy(-6)
				},
			},
		},
		{
			name: "weighted-terms-custom-weights",
			names: []string{
				"dist-source-linux-amd64.tar.gz",
				"dist-linux-amd64.tar.gz",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{"gz"},
				WeightedTerms: map[string]int{
					"source": -20,
				},
			},
			expected: []Sorted{
				{
					Key:   "dist-linux-amd64.tar.gz",
					Value: 89, // os(40) + arch(30) + ext(20) + accuracy(-1)
				},
				{
					Key:   "dist-source-linux-amd64.tar.gz",
					Value: 64, // os(40) + arch(30) + ext(20) + weighted(-20) + accuracy(-6)
				},
			},
		},
		{
			name: "mac-false-positive-in-machine",
			names: []string{
				"machine-linux-amd64",
				"tool-darwin-amd64",
			},
			opts: &Options{
				OS:        []string{"darwin", "mac", "macos", "osx"},
				Arch:      []string{"amd64"},
				InvalidOS: []string{"linux", "windows"},
			},
			expected: []Sorted{
				{
					Key:   "tool-darwin-amd64",
					Value: 69, // os-"darwin"(40) + arch(30) + accuracy(-1)
				},
				{
					// FIXED: "mac" no longer matches inside "machine" with segment matching
					Key:   "machine-linux-amd64",
					Value: -18, // no OS match + arch(30) + invalidOS-"linux"(-40) + accuracy(-8)
				},
			},
		},
		{
			name: "win-false-positive-in-winding",
			names: []string{
				"winding-linux-amd64",
				"tool-windows-amd64.exe",
			},
			opts: &Options{
				OS:         []string{"windows", "win"},
				Arch:       []string{"amd64"},
				Extensions: []string{"exe"},
				InvalidOS:  []string{"linux", "darwin"},
			},
			expected: []Sorted{
				{
					Key:   "tool-windows-amd64.exe",
					Value: 89, // os-"windows"(40) + arch(30) + ext(20) + accuracy(-1)
				},
				{
					// FIXED: "win" no longer matches inside "winding" with segment matching
					Key:   "winding-linux-amd64",
					Value: -18, // no OS match + arch(30) + invalidOS-"linux"(-40) + accuracy(-8)
				},
			},
		},
		{
			name: "multi-segment-arch-x86-64",
			names: []string{
				"tool-linux-x86-64.tar.gz",
				"tool-linux-aarch64.tar.gz",
			},
			opts: &Options{
				OS:          []string{"linux"},
				Arch:        []string{"amd64", "x86_64", "x86-64"},
				Extensions:  []string{"gz"},
				InvalidArch: []string{"aarch64", "arm64"},
			},
			expected: []Sorted{
				{
					Key:   "tool-linux-x86-64.tar.gz",
					Value: 89, // os(40) + arch-"x86-64"(30) + ext(20) + accuracy(-1)
				},
				{
					Key:   "tool-linux-aarch64.tar.gz",
					Value: 22, // os(40) + invalidarch-"aarch64"(-30) + ext(20) + accuracy(-8)
				},
			},
		},
		{
			name: "accuracy-penalty-unknown-segments",
			names: []string{
				"fancy-tool-linux-amd64",
				"fancy-tool-extra-metadata-linux-amd64",
			},
			opts: &Options{
				OS:         []string{"linux"},
				Arch:       []string{"amd64"},
				Extensions: []string{""},
				Terms:      []string{"fancy-tool"},
			},
			expected: []Sorted{
				{
					Key:   "fancy-tool-linux-amd64",
					Value: 86, // os(40) + arch(30) + term(10) + accuracy(+6: fancy-tool+2, linux+2, amd64+2)
				},
				{
					Key:   "fancy-tool-extra-metadata-linux-amd64",
					Value: 76, // os(40) + arch(30) + term(10) + accuracy(-4: fancy-tool+2, extra-5, metadata-5, linux+2, amd64+2)
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
