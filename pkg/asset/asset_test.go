package asset

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/dsnet/compress/bzip2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/ulikunitz/xz"
)

func init() {
	logrus.SetLevel(logrus.TraceLevel)
}

func TestAsset(t *testing.T) {
	cases := []struct {
		name        string
		displayName string
		expectType  Type
	}{
		{"test", "Test", Unknown},
		{"test.tar.gz", "Test", Archive},
		{"test.tar.gz.asc", "Test", Signature},
		{"dist.tar.gz.sig", "dist.tar.gz.sig", Signature},
	}

	for _, c := range cases {
		asset := New(c.name, c.displayName, "linux", "amd64", "1.0.0")

		if asset.GetName() != c.name {
			t.Errorf("expected name to be %s, got %s", c.name, asset.GetName())
		}
		if asset.GetDisplayName() != c.displayName {
			t.Errorf("expected display name to be %s, got %s", c.displayName, asset.GetDisplayName())
		}
		if asset.Type != c.expectType {
			t.Errorf("expected type to be %d, got %d", c.expectType, asset.Type)
		}
	}
}

func TestAssetDefaults(t *testing.T) {
	asset := New("dist-linux-amd64.tar.gz", "dist-linux-amd64.tar.gz", "linux", "amd64", "1.0.0")
	err := asset.Download(context.TODO())
	assert.Error(t, err)

	assert.Equal(t, Archive, asset.GetType())
	assert.Equal(t, "not-implemented", asset.ID())
	assert.Equal(t, "dist-linux-amd64.tar.gz", asset.GetDisplayName())
	assert.Equal(t, "dist-linux-amd64.tar.gz", asset.GetName())
	assert.Equal(t, "", asset.GetFilePath())
	assert.Equal(t, "", asset.GetTempPath())
	assert.Equal(t, asset, asset.GetAsset())
	assert.Equal(t, make([]*File, 0), asset.GetFiles())
}

func TestAssetTypes(t *testing.T) {
	cases := []struct {
		name     string
		fileType Type
	}{
		{
			name:     "dist-linux-amd64.deb",
			fileType: Installer,
		},
		{
			name:     "dist-linux-amd64.rpm",
			fileType: Installer,
		},
		{
			name:     "dist-linux-amd64.tar.gz",
			fileType: Archive,
		},
		{
			name:     "dist-linux-amd64.exe",
			fileType: Binary,
		},
		{
			name:     "dist-linux-amd64",
			fileType: Unknown,
		},
		{
			name:     "dist-linux-amd64.tar.gz.sig",
			fileType: Signature,
		},
		{
			name:     "dist-linux-amd64.tar.gz.pem",
			fileType: Key,
		},
		{
			name:     "checksums.txt",
			fileType: Checksum,
		},
		{
			name:     "dist-linux.SHASUMS",
			fileType: Checksum,
		},
		{
			name:     "dist-linux-amd64.tar.gz.sha256",
			fileType: Checksum,
		},
		{
			name:     "dist-linux.nse",
			fileType: Unknown,
		},
		{
			name:     "dist-linux.deb",
			fileType: Installer,
		},
		{
			name:     "dist-windows.msi",
			fileType: Installer,
		},
		{
			name:     "dist-linux-amd64.sbom.json",
			fileType: SBOM,
		},
		{
			name:     "dist-linux-amd64.json",
			fileType: Data,
		},
		{
			name:     "dist-linux-amd64.sbom",
			fileType: SBOM,
		},
	}

	for _, c := range cases {
		asset := New(c.name, c.name, "linux", "amd64", "1.0.0")
		assert.Equal(t, c.fileType, asset.GetType(), fmt.Sprintf("expected type to be %d, got %d for %s", c.fileType, asset.GetType(), c.name))
	}
}

type internalFile struct {
	name    string
	mode    int64
	content []byte
}

func TestAssetExtract(t *testing.T) {
	cases := []struct {
		name          string
		fileType      Type
		downloadFile  string
		expectedFiles []string
		expectError   bool
	}{
		{
			name:     "dist-linux-amd64.tar.gz",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "test-file",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"test-file",
			},
		},
		{
			name:         "dist-linux-amd64.zip",
			fileType:     Archive,
			downloadFile: createZip(t, "test-file", []byte{0x7F, 0x45, 0x4C, 0x46}),
			expectedFiles: []string{
				"test-file",
				"docs/readme.md",
			},
		},
		{
			name:     "dist-linux-amd64.tar.bz2",
			fileType: Archive,
			downloadFile: createTarBz2(t, []internalFile{
				{
					name:    "test-file",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"test-file",
			},
		},
		{
			name:     "dist-linux-amd64.tar.xz",
			fileType: Archive,
			downloadFile: createTarXz(t, []internalFile{
				{
					name:    "test-file",
					mode:    0644,
					content: []byte("This is a test file content"),
				},
			}),
			expectedFiles: []string{
				"test-file",
			},
		},
		{
			name:         "dist-linux-amd64",
			fileType:     Binary,
			downloadFile: createFile(t, []byte("This is a test file content")),
			expectedFiles: []string{
				"dist-linux-amd64",
			},
		},
		{
			name:         "windows-executable",
			fileType:     Binary,
			downloadFile: createFile(t, []byte("This is a test file content")),
			expectedFiles: []string{
				"windows-executable",
			},
		},
		{
			name:     "dist-linux-multi-amd64.tar.gz",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "bin1",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
				{
					name:    "bin2",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
				{
					name:    "docs/readme.md",
					mode:    0600,
					content: []byte("this is a readme"),
				},
			}),
			expectedFiles: []string{
				"bin1",
				"bin2",
				"docs/readme.md",
			},
		},
		{
			name:         "empty.zip",
			fileType:     Archive,
			downloadFile: createEmptyZip(t),
			expectError:  true,
		},
		{
			name:         "empty.tar.gz",
			fileType:     Archive,
			downloadFile: createTarGz(t, []internalFile{}),
			expectedFiles: []string{
				"test-*.tar.gz",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			asset := New(c.name, c.name, "linux", "amd64", "1.0.0")
			asset.DownloadPath = c.downloadFile

			defer func(asset *Asset) {
				_ = asset.Cleanup()
			}(asset)

			defer func(path string) {
				_ = os.RemoveAll(path)
			}(c.downloadFile)

			err := asset.Extract()
			if c.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Len(t, asset.Files, len(c.expectedFiles))

			for i, f := range asset.Files {
				if slices.Contains(c.expectedFiles, f.Name) {
					assert.Equal(t, c.expectedFiles[i], f.Name)
				}
			}
		})
	}
}

func TestAssetInstall(t *testing.T) {
	cases := []struct {
		name          string
		os            string
		arch          string
		version       string
		fileType      Type
		downloadFile  string
		expectedFiles []string
		expectError   bool
	}{
		{
			name:     "dist-linux-amd64.tar.gz",
			os:       "linux",
			arch:     "amd64",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "test-binary",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"test-binary",
			},
		},
		{
			name: "dist-darwin-amd64.tar.gz",
			os:   "darwin",
			arch: "amd64",
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "test-binary",
					mode:    0755,
					content: []byte{0xFE, 0xED, 0xFA, 0xCE},
				},
			}),
			expectedFiles: []string{
				"test-binary",
			},
		},
		{
			name:         "dist-linux-amd64.zip",
			os:           "linux",
			arch:         "amd64",
			fileType:     Archive,
			downloadFile: createZip(t, "test-binary", []byte{0x7F, 0x45, 0x4C, 0x46}),
			expectedFiles: []string{
				"test-binary",
			},
		},
		{
			name:     "dist-linux-amd64.tar.bz2",
			os:       "linux",
			arch:     "amd64",
			fileType: Archive,
			downloadFile: createTarBz2(t, []internalFile{
				{
					name:    "test-binary",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"test-binary",
			},
		},
		{
			name:     "dist-linux-amd64.tar.xz",
			os:       "linux",
			arch:     "amd64",
			fileType: Archive,
			downloadFile: createTarXz(t, []internalFile{
				{
					name:    "test-binary",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"test-binary",
			},
		},
		{
			name:         "dist-darwin-amd64",
			os:           "darwin",
			arch:         "amd64",
			fileType:     Binary,
			downloadFile: createFile(t, []byte{0xFE, 0xED, 0xFA, 0xCE}),
			expectedFiles: []string{
				"dist",
			},
		},
		{
			name:         "test.exe",
			os:           "windows",
			arch:         "amd64",
			fileType:     Binary,
			downloadFile: createFile(t, []byte{0x4D, 0x5A}),
			expectedFiles: []string{
				"test.exe",
			},
		},
		{
			name:     "dist-linux-multi-amd64.tar.gz",
			os:       "linux",
			arch:     "amd64",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "bin1",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
				{
					name:    "bin2",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
				{
					name:    "docs/readme.md",
					mode:    0600,
					content: []byte("this is a readme"),
				},
			}),
			expectedFiles: []string{
				"bin1",
				"bin2",
			},
		},
		{
			name:     "dist-v2.25.0-linux-amd64.tar.gz",
			os:       "linux",
			arch:     "amd64",
			version:  "2.25.0",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "dist-v2.25.0-linux-amd64",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"dist",
			},
		},
		{
			name:     "dist-2.25.0-linux-amd64.tar.gz",
			os:       "linux",
			arch:     "amd64",
			version:  "2.25.0",
			fileType: Archive,
			downloadFile: createTarGz(t, []internalFile{
				{
					name:    "dist-2.25.0-linux-amd64",
					mode:    0755,
					content: []byte{0x7F, 0x45, 0x4C, 0x46},
				},
			}),
			expectedFiles: []string{
				"dist",
			},
		},
		{
			name:         "test-1.12.3-darwin-10.12-amd64",
			os:           "darwin",
			arch:         "amd64",
			version:      "1.12.3",
			fileType:     Binary,
			downloadFile: createFile(t, []byte{0xFE, 0xED, 0xFA, 0xCE}),
			expectedFiles: []string{
				"test",
			},
		},
		{
			name:         "delta-aarch64-apple-darwin",
			os:           "darwin",
			arch:         "arm64",
			version:      "1.0.0",
			fileType:     Binary,
			downloadFile: createFile(t, []byte{0xFE, 0xED, 0xFA, 0xCE}),
			expectedFiles: []string{
				"delta",
			},
		},
		{
			name:         "delta-x86_64-apple-darwin",
			os:           "darwin",
			arch:         "amd64",
			version:      "1.0.0",
			fileType:     Binary,
			downloadFile: createFile(t, []byte{0xFE, 0xED, 0xFA, 0xCE}),
			expectedFiles: []string{
				"delta",
			},
		},
		{
			name:     "pie-compiled-binary",
			os:       "linux",
			arch:     "amd64",
			version:  "1.0.0",
			fileType: Binary,
			downloadFile: createFile(t, []byte{
				0x7F, 0x45, 0x4C, 0x46, // Magic number "\x7FELF"
				0x02,                                     // Class: 64-bit
				0x01,                                     // Data: Little-endian
				0x01,                                     // Version: ELF current
				0x00,                                     // OS/ABI: System V
				0x00,                                     // ABI Version
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Padding
				0x03, 0x00, // Type: ET_DYN (Shared object file)
				0x3E, 0x00, // Machine: x86-64 (AMD64)
				0x01, 0x00, 0x00, 0x00, // Version: Current
				// Entry point address, Program header table offset, Section header table offset:
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Entry point address (placeholder)
				0x40, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Start of program headers
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Start of section headers
				0x00, 0x00, 0x00, 0x00, // Flags
				0x40, 0x00, // ELF header size (64 bytes)
				0x38, 0x00, // Program header size
				0x01, 0x00, // Number of program headers
				0x40, 0x00, // Section header size
				0x00, 0x00, // Number of section headers
				0x00, 0x00, // Section header string table index
			}),
			expectedFiles: []string{
				"pie-compiled-binary",
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Create a temporary directory for the binary installation
			binDir, err := os.MkdirTemp("", "bin")
			assert.NoError(t, err)
			defer os.RemoveAll(binDir)

			optDir, err := os.MkdirTemp("", "opt")
			assert.NoError(t, err)
			defer os.RemoveAll(optDir)

			version := c.version
			if version == "" {
				version = "1.0.0"
			}

			asset := New(c.name, c.name, c.os, c.arch, version)
			asset.DownloadPath = c.downloadFile

			err = asset.Extract()
			assert.NoError(t, err)

			err = asset.Install("test-id", binDir, optDir)
			assert.NoError(t, err)

			for _, fileName := range c.expectedFiles {
				destBinaryName := filepath.Base(fileName)
				destBinPath := filepath.Join(optDir, destBinaryName)

				baseLinkName := filepath.Join(binDir, filepath.Base(fileName))
				versionedLinkName := filepath.Join(binDir, fmt.Sprintf("%s@%s", filepath.Base(fileName), version))

				_, err = os.Stat(destBinPath)
				assert.NoError(t, err)

				if c.os == runtime.GOOS && c.arch == runtime.GOARCH {
					_, err = os.Stat(baseLinkName)
					assert.NoError(t, err)

					linkPath, err := os.Readlink(baseLinkName)
					assert.NoError(t, err)
					assert.Equal(t, destBinaryName, filepath.Base(linkPath))

					_, err = os.Stat(versionedLinkName)
					assert.NoError(t, err)

					linkPath, err = os.Readlink(versionedLinkName)
					assert.NoError(t, err)
					assert.Equal(t, destBinaryName, filepath.Base(linkPath))
				}
			}

			_ = filepath.Walk(binDir, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				fmt.Println(">", path)

				return nil
			})
		})
	}
}

// -- helper functions below --

// createEmptyZip creates an empty zip file
func createEmptyZip(t *testing.T) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-empty-*.zip")
	assert.NoError(t, err)
	defer tmpFile.Close()

	// Create a zip writer
	zw := zip.NewWriter(tmpFile)
	defer zw.Close()

	return tmpFile.Name()
}

// createFile creates a temporary file with the given content
func createFile(t *testing.T, content []byte) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*")
	assert.NoError(t, err)
	defer tmpFile.Close()

	_, err = tmpFile.Write(content)
	assert.NoError(t, err)

	return tmpFile.Name()
}

// createTar creates a tar archive with the given files
func createTar(t *testing.T, out io.Writer, files []internalFile) error {
	t.Helper()

	// Create a tar writer
	tw := tar.NewWriter(out)
	defer tw.Close()

	for _, f := range files {
		parts := strings.Split(f.name, "/")
		if len(parts) > 1 {
			for i := 0; i < len(parts)-1; i++ {
				// Add a directory to the tar archive
				dirHdr := &tar.Header{
					Name: parts[0] + "/",
					Mode: 0755,
					Size: 0,
				}
				err := tw.WriteHeader(dirHdr)
				if err != nil {
					return err
				}
			}
		}

		// Add a file to the tar archive
		hdr := &tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.content)),
		}
		err := tw.WriteHeader(hdr)
		if err != nil {
			return err
		}

		_, err = tw.Write(f.content)
		if err != nil {
			return err
		}
	}

	return nil
}

// createTarGz creates a tar.gz archive with the given files
func createTarGz(t *testing.T, files []internalFile) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.tar.gz")
	assert.NoError(t, err)
	defer tmpFile.Close()

	// Create a gzip writer
	gw := gzip.NewWriter(tmpFile)
	defer gw.Close()

	err = createTar(t, gw, files)
	assert.NoError(t, err)

	return tmpFile.Name()
}

// createTarBz2 creates a tar.bz2 archive with the given files
func createTarBz2(t *testing.T, files []internalFile) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.tar.bz2")
	assert.NoError(t, err)
	defer tmpFile.Close()

	// Create a bzip2 writer
	bw, err := bzip2.NewWriter(tmpFile, &bzip2.WriterConfig{Level: bzip2.BestCompression})
	assert.NoError(t, err)
	defer bw.Close()

	err = createTar(t, bw, files)
	assert.NoError(t, err)

	return tmpFile.Name()
}

// createTarXz creates a tar.xz archive with the given files
func createTarXz(t *testing.T, files []internalFile) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.tar.xz")
	assert.NoError(t, err)
	defer tmpFile.Close()

	// Create a xz writer
	xw, err := xz.NewWriter(tmpFile)
	assert.NoError(t, err)
	defer xw.Close()

	err = createTar(t, xw, files)
	assert.NoError(t, err)

	return tmpFile.Name()
}

// createZip creates a zip archive with the given content
func createZip(t *testing.T, fileName string, content []byte) string {
	t.Helper()

	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "test-*.zip")
	assert.NoError(t, err)
	defer tmpFile.Close()

	// Create a zip writer
	zw := zip.NewWriter(tmpFile)
	defer zw.Close()

	// Add a file to the zip archive
	w, err := zw.Create(fileName)
	assert.NoError(t, err)

	_, err = io.Copy(w, bytes.NewReader(content))
	assert.NoError(t, err)

	// Add docs/ directory to the zip archive
	_, err = zw.Create("docs/")
	assert.NoError(t, err)

	// Add README.md file to the docs/ directory
	readmeContent := "This is a README file."
	w, err = zw.Create("docs/README.md")
	assert.NoError(t, err)

	_, err = io.Copy(w, bytes.NewReader([]byte(readmeContent)))
	assert.NoError(t, err)

	return tmpFile.Name()
}
