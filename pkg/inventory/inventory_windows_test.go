package inventory_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ekristen/distillery/pkg/config"
	"github.com/ekristen/distillery/pkg/inventory"
)

func TestInventoryWindowsNew(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "inventory_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir)
	cfg, err := config.New("")
	if err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	symPath := filepath.ToSlash(filepath.Join(tempDir, ".distillery", "bin"))
	binPath := filepath.ToSlash(filepath.Join(tempDir, ".distillery", "opt"))
	_ = os.MkdirAll(symPath, 0755)
	_ = os.MkdirAll(binPath, 0755)

	cfg.Path = filepath.ToSlash(filepath.Join(tempDir, ".distillery"))
	cfg.BinPath = binPath

	symlinks := map[string]string{
		"test":               "github/ekristen/test/2.0.0/test",
		"test@1.0.0":         "github/ekristen/test/1.0.0/test",
		"test@2.0.0":         "github/ekristen/test/2.0.0/test",
		"another-test@3.0.0": "github/ekristen/another-test/3.0.0/another-test",
		"another-test@3.0.1": "github/ekristen/another-test/3.0.1/another-test",
	}

	for link, bin := range symlinks {
		targetBase := filepath.Dir(bin)
		targetName := filepath.Base(bin)
		realBin := filepath.ToSlash(filepath.Join(binPath, targetBase, targetName))
		_ = os.MkdirAll(filepath.Join(realBin, targetBase), 0755)
		_ = os.WriteFile(realBin, []byte("test"), 0600)

		symlinkPath := filepath.ToSlash(filepath.Join(symPath, link))
		if err := os.Symlink(realBin, symlinkPath); err != nil {
			t.Fatalf("Failed to create symlink: %v", err)
		}
	}

	dirFS := os.DirFS(tempDir)
	inv := inventory.New(dirFS, tempDir, ".distillery/bin", cfg)

	assert.NotNil(t, inv)
	assert.Equal(t, 2, inv.Count())
	assert.Equal(t, 4, inv.FullCount()) // Note: 4 because 1 is marked as latest

	binVersionsExpected := &inventory.Bin{
		Name:   "test",
		Source: "github",
		Owner:  "ekristen",
		Repo:   "test",
		Versions: []*inventory.Version{
			{
				Version: "1.0.0",
				Path:    ".distillery/bin/test@1.0.0",
				Target: filepath.ToSlash(
					filepath.Join(tempDir, ".distillery", "opt", "github", "ekristen", "test", "1.0.0", "test"),
				),
			},
			{
				Version: "2.0.0",
				Path:    ".distillery/bin/test@2.0.0",
				Latest:  true,
				Target: filepath.ToSlash(
					filepath.Join(tempDir, ".distillery", "opt", "github", "ekristen", "test", "2.0.0", "test"),
				),
			},
		},
	}

	assert.EqualValues(t, binVersionsExpected, inv.GetBinVersions("github/ekristen/test"))

	binVersionExpected := &inventory.Version{
		Version: "1.0.0",
		Path:    ".distillery/bin/test@1.0.0",
		Target: filepath.ToSlash(
			filepath.Join(tempDir, ".distillery", "opt", "github", "ekristen", "test", "1.0.0", "test"),
		),
	}

	assert.EqualValues(t, binVersionExpected, inv.GetBinVersion("github/ekristen/test", "1.0.0"))

	latestBinVersionExpected := &inventory.Version{
		Version: "2.0.0",
		Path:    ".distillery/bin/test@2.0.0",
		Latest:  true,
		Target: filepath.ToSlash(
			filepath.Join(tempDir, ".distillery", "opt", "github", "ekristen", "test", "2.0.0", "test"),
		),
	}

	assert.EqualValues(t, latestBinVersionExpected, inv.GetLatestVersion("github/ekristen/test"))
}

func TestInventoryWindowsAddVersion(t *testing.T) {
	cases := []struct {
		name     string
		bins     map[string]string
		expected map[string]*inventory.Bin
	}{
		{
			name: "simple",
			bins: map[string]string{
				"c:/users/test/.distillery/bin/test@1.0.0": "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
			},
			expected: map[string]*inventory.Bin{
				"github/ekristen/test": {
					Name:   "test",
					Source: "github",
					Owner:  "ekristen",
					Repo:   "test",
					Versions: []*inventory.Version{
						{
							Version: "1.0.0",
							Path:    "c:/users/test/.distillery/bin/test@1.0.0",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
						},
					},
				},
			},
		},
		{
			name: "multiple",
			bins: map[string]string{
				"c:/users/test/.distillery/bin/test@1.0.0": "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
				"c:/users/test/.distillery/bin/test@2.0.0": "c:/users/test/.distillery/opt/github/ekristen/test/2.0.0/test",
			},
			expected: map[string]*inventory.Bin{
				"github/ekristen/test": {
					Name:   "test",
					Source: "github",
					Owner:  "ekristen",
					Repo:   "test",
					Versions: []*inventory.Version{
						{
							Version: "1.0.0",
							Path:    "c:/users/test/.distillery/bin/test@1.0.0",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
						},
						{
							Version: "2.0.0",
							Path:    "c:/users/test/.distillery/bin/test@2.0.0",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/test/2.0.0/test",
						},
					},
				},
			},
		},
		{
			name: "complex",
			bins: map[string]string{
				"c:/users/test/.distillery/bin/test@1.0.0":         "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
				"c:/users/test/.distillery/bin/test@1.0.1":         "c:/users/test/.distillery/opt/github/ekristen/test/1.0.1/test",
				"c:/users/test/.distillery/bin/another-test@1.0.0": "c:/users/test/.distillery/opt/github/ekristen/another-test/1.0.0/another-test",
				"c:/users/test/.distillery/bin/another-test@1.0.1": "c:/users/test/.distillery/opt/github/ekristen/another-test/1.0.1/another-test",
			},
			expected: map[string]*inventory.Bin{
				"github/ekristen/test": {
					Name:   "test",
					Source: "github",
					Owner:  "ekristen",
					Repo:   "test",
					Versions: []*inventory.Version{
						{
							Version: "1.0.0",
							Path:    "c:/users/test/.distillery/bin/test@1.0.0",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/test/1.0.0/test",
						},
						{
							Version: "1.0.1",
							Path:    "c:/users/test/.distillery/bin/test@1.0.1",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/test/1.0.1/test",
						},
					},
				},
				"github/ekristen/another-test": {
					Name:   "another-test",
					Source: "github",
					Owner:  "ekristen",
					Repo:   "another-test",
					Versions: []*inventory.Version{
						{
							Version: "1.0.0",
							Path:    "c:/users/test/.distillery/bin/another-test@1.0.0",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/another-test/1.0.0/another-test",
						},
						{
							Version: "1.0.1",
							Path:    "c:/users/test/.distillery/bin/another-test@1.0.1",
							Target:  "c:/users/test/.distillery/opt/github/ekristen/another-test/1.0.1/another-test",
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg, err := config.New("")
			cfg.Path = "c:/users/test/.distillery"
			cfg.BinPath = "c:/users/test/.distillery/opt"

			assert.NoError(t, err)
			inv := inventory.Inventory{}
			inv.SetConfig(cfg)
			for bin, target := range tc.bins {
				_ = inv.AddVersion(filepath.ToSlash(bin), filepath.ToSlash(target))
			}

			for bin, expected := range tc.expected {
				assert.ElementsMatch(t, expected.Versions, inv.Bins[bin].Versions)
			}
		})
	}
}

func BenchmarkInventoryWindowsNew(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "inventory_test")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func(path string) {
		_ = os.RemoveAll(path)
	}(tempDir) // Clean up the temp directory after the test
	cfg, err := config.New("")
	if err != nil {
		b.Fatalf("Failed to create config: %v", err)
	}

	symPath := filepath.Join(tempDir, ".distillery", "bin")
	binPath := filepath.Join(tempDir, ".distillery", "opt")
	_ = os.MkdirAll(symPath, 0755)
	_ = os.MkdirAll(binPath, 0755)

	// Generate fake binary files to simulate version binaries
	binaries := map[string]string{
		"test":               "github/ekristen/test/2.0.0/test",
		"test@1.0.0":         "github/ekristen/test/1.0.0/test",
		"test@2.0.0":         "github/ekristen/test/2.0.0/test",
		"another-test@1.0.0": "github/ekristen/another-test/1.0.0/another-test",
		"another-test@1.0.1": "github/ekristen/another-test/1.0.1/another-test",
	}

	for bin, target := range binaries {
		targetBase := filepath.Dir(target)
		targetName := filepath.Base(target)
		realBin := filepath.Join(binPath, targetBase, targetName)
		_ = os.MkdirAll(filepath.Join(realBin, targetBase), 0755)
		_ = os.WriteFile(realBin, []byte("test"), 0600)

		symlinkPath := filepath.Join(symPath, bin)
		if err := os.Symlink(realBin, symlinkPath); err != nil {
			b.Fatalf("Failed to create symlink: %v", err)
		}
	}

	dirFS := os.DirFS(tempDir)

	b.ResetTimer() // Reset the timer to exclude setup time

	for i := 0; i < b.N; i++ {
		_ = inventory.New(dirFS, tempDir, ".distillery/bin", cfg)
	}
}

func BenchmarkInventoryWindowsHomeDir(b *testing.B) {
	userDir, _ := os.UserHomeDir()
	basePath := "C:/"
	baseFS := os.DirFS(basePath)
	binPath := filepath.Join(userDir, ".distillery", "bin")
	cfg, err := config.New("")
	if err != nil {
		b.Fatalf("Failed to create config: %v", err)
	}

	for i := 0; i < b.N; i++ {
		_ = inventory.New(baseFS, basePath, binPath, cfg)
	}
}
