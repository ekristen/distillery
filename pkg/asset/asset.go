package asset

import (
	"archive/tar"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/ProtonMail/gopenpgp/v2/crypto"
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/krolaw/zipstream"
	"github.com/sirupsen/logrus"
	"github.com/xi2/xz"

	"github.com/ekristen/distillery/pkg/common"
	"github.com/ekristen/distillery/pkg/osconfig"
)

var (
	msiType      = filetype.AddType("msi", "application/octet-stream")
	apkType      = filetype.AddType("apk", "application/vnd.android.package-archive")
	ascType      = filetype.AddType("asc", "text/plain")
	pemType      = filetype.AddType("pem", "application/x-pem-file")
	certType     = filetype.AddType("cert", "application/x-x509-ca-cert")
	crtType      = filetype.AddType("crt", "application/x-x509-ca-cert")
	sigType      = filetype.AddType("sig", "text/plain")
	pkgType      = filetype.AddType("pkg", "application/octet-stream")
	sbomJSONType = filetype.AddType("sbom.json", "application/json")
	bomJSONType  = filetype.AddType("bom.json", "application/json")
	jsonType     = filetype.AddType("json", "application/json")
	sbomType     = filetype.AddType("sbom", "application/octet-stream")
	bomType      = filetype.AddType("bom", "application/octet-stream")
	pubType      = filetype.AddType("pub", "text/plain")
	tarGzType    = filetype.AddType("tgz", "application/tar+gzip")
	zstdType     = filetype.AddType("zst", "application/zstd")

	ignoreFileExtensions = []string{
		".txt",
		".sbom",
		".json",
	}

	executableMimetypes = []string{
		"application/x-mach-binary",
		"application/x-executable",
		"application/x-elf",
		"application/vnd.microsoft.portable-executable",
	}
)

// Type is the type of asset
type Type int

func (t Type) String() string {
	return [...]string{"unknown", "archive", "binary", "installer", "checksum", "signature", "key", "sbom", "data"}[t]
}

const (
	Unknown Type = iota
	Archive
	Binary
	Installer
	Checksum
	Signature
	Key
	SBOM
	Data

	ChecksumTypeNone  = "none"
	ChecksumTypeFile  = "single"
	ChecksumTypeMulti = "multi"
)

// processorFunc is a function that processes a reader
type processorFunc func(io.Reader) (io.Reader, error)

// New creates a new asset
func New(name, displayName, osName, osArch, version string) *Asset {
	a := &Asset{
		Name:        name,
		DisplayName: displayName,
		OS:          osName,
		Arch:        osArch,
		Version:     version,
		Files:       make([]*File, 0),
	}

	a.Type = a.Classify(name)

	if a.Type == Key || a.Type == Signature || a.Type == Checksum {
		parentName := strings.ReplaceAll(name, filepath.Ext(name), "")
		parentName = strings.TrimSuffix(parentName, "-keyless")

		a.ParentType = a.Classify(parentName)
	}

	return a
}

type File struct {
	Name        string
	Alias       string
	Installable bool
}

type Asset struct {
	Name         string
	DisplayName  string
	Type         Type
	ParentType   Type
	ChecksumType string
	MatchedAsset IAsset

	OS      string
	Arch    string
	Version string

	Extension    string
	DownloadPath string
	Hash         string
	TempDir      string
	Files        []*File
}

func (a *Asset) ID() string {
	return "not-implemented"
}
func (a *Asset) Path() string { return "not-implemented" }

func (a *Asset) GetName() string {
	return a.Name
}

func (a *Asset) GetDisplayName() string {
	return a.DisplayName
}

func (a *Asset) GetType() Type {
	return a.Type
}
func (a *Asset) GetParentType() Type {
	return a.ParentType
}
func (a *Asset) GetChecksumType() string {
	name := strings.ToLower(a.Name)
	if strings.HasSuffix(name, ".sha512") ||
		strings.HasSuffix(name, ".sha512sum") ||
		strings.HasSuffix(name, ".sha256") ||
		strings.HasSuffix(name, ".sha256sum") ||
		strings.HasSuffix(name, ".md5") ||
		strings.HasSuffix(name, ".md5sum") ||
		strings.HasSuffix(name, ".sha1") ||
		strings.HasSuffix(name, ".sha1sum") ||
		strings.HasSuffix(name, ".shasum") {
		return ChecksumTypeFile
	}
	if strings.Contains(name, "checksums") ||
		strings.Contains(name, "checksum") {
		return ChecksumTypeMulti
	}
	if strings.Contains(name, "sha") &&
		strings.Contains(name, "sums") {
		return ChecksumTypeMulti
	} else if strings.Contains(name, "sums") {
		return ChecksumTypeMulti
	}
	return ChecksumTypeNone
}

func (a *Asset) GetMatchedAsset() IAsset {
	return a.MatchedAsset
}
func (a *Asset) SetMatchedAsset(asset IAsset) {
	a.MatchedAsset = asset
}

func (a *Asset) GetAsset() *Asset {
	return a
}

func (a *Asset) GetFiles() []*File {
	return a.Files
}
func (a *Asset) GetTempPath() string {
	return a.TempDir
}

func (a *Asset) Download(_ context.Context) error {
	return fmt.Errorf("not implemented")
}

func (a *Asset) GetFilePath() string {
	return a.DownloadPath
}

// Classify determines the type of asset based on the file extension
func (a *Asset) Classify(name string) Type { //nolint:gocyclo
	aType := Unknown

	if ext := strings.TrimPrefix(filepath.Ext(name), "."); ext != "" {
		switch filetype.GetType(ext) {
		case matchers.TypeDeb, matchers.TypeRpm, msiType, apkType, pkgType:
			aType = Installer
		case matchers.TypeGz, matchers.TypeZip, matchers.TypeXz, matchers.TypeTar, matchers.TypeBz2, tarGzType, zstdType, matchers.TypeZstd:
			aType = Archive
		case matchers.TypeExe:
			aType = Binary
		case sigType, ascType:
			aType = Signature
		case pemType, pubType, certType, crtType:
			aType = Key
		case sbomJSONType, bomJSONType, sbomType, bomType:
			aType = SBOM
		case jsonType:
			aType = Data

			if strings.Contains(name, ".sbom") || strings.Contains(name, ".bom") {
				aType = SBOM
			}
		default:
			aType = Unknown
		}
	}

	if aType == Unknown {
		logrus.Tracef("classifying asset based on name: %s", name)
		name = strings.ToLower(name)
		if strings.HasSuffix(name, ".sha512") ||
			strings.HasSuffix(name, ".sha512sum") ||
			strings.HasSuffix(name, ".sha256") ||
			strings.HasSuffix(name, ".sha256sum") ||
			strings.HasSuffix(name, ".md5") ||
			strings.HasSuffix(name, ".md5sum") ||
			strings.HasSuffix(name, ".sha1") ||
			strings.HasSuffix(name, ".sha1sum") ||
			strings.HasSuffix(name, ".shasum") {
			aType = Checksum
		}
		if strings.Contains(name, "checksums") {
			aType = Checksum
		}
		if strings.Contains(name, "sha") && strings.Contains(name, "sums") {
			aType = Checksum
		} else if strings.Contains(name, "sums") {
			aType = Checksum
		}
	}

	if aType == Unknown {
		if strings.Contains(name, "-pivkey-") {
			aType = Key
		} else if strings.Contains(name, "pkcs") && strings.Contains(name, "key") {
			aType = Key
		}
	}

	logrus.Tracef("classified: %s - %s (type: %d)", name, aType, aType)

	return aType
}

func (a *Asset) copyFile(srcFile, dstFile string) error {
	// Open the source file for reading
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	defer dst.Close()

	// Copy the contents of the source file to the destination file
	_, err = io.Copy(dst, src)
	if err != nil {
		return err
	}

	return nil
}

// determineInstallable determines if the file is installable or not based on the mimetype
func (a *Asset) determineInstallable() {
	logrus.Tracef("files to process: %d", len(a.Files))
	for _, file := range a.Files {
		// Actual path to the downloaded/extracted file
		fullPath := filepath.Join(a.TempDir, file.Name)

		logrus.Debug("checking file for installable: ", file.Name)
		m, err := mimetype.DetectFile(fullPath)
		if err != nil {
			logrus.WithError(err).Warn("unable to determine mimetype")
		}

		logrus.Debug("found mimetype: ", m.String())

		if slices.Contains(ignoreFileExtensions, m.Extension()) {
			logrus.Tracef("ignoring file: %s", file.Name)
			continue
		}

		if slices.Contains(executableMimetypes, m.String()) {
			logrus.Debugf("found installable executable: %s, %s, %s", file.Name, m.String(), m.Extension())
			file.Installable = true
		}
	}
}

var versionReplace = regexp.MustCompile(`\d+\.\d+`)

// Install installs the asset
// TODO(ek): simplify this function
func (a *Asset) Install(id, binDir, optDir string) error {
	found := false

	if err := os.MkdirAll(optDir, 0755); err != nil {
		return err
	}

	a.determineInstallable()

	for _, file := range a.Files {
		if !file.Installable {
			logrus.Tracef("skipping file: %s", file.Name)
			continue
		}

		found = true
		logrus.Debugf("installing file: %s", file.Name)

		fullPath := filepath.Join(a.TempDir, file.Name)
		dstFilename := filepath.Base(fullPath)
		if file.Alias != "" {
			dstFilename = file.Alias
		}

		logrus.Trace("pre-dstFilename: ", dstFilename)

		// Strip the OS and Arch from the filename if it exists, this happens mostly when the binary is being
		// uploaded directly instead of being encapsulated in a tarball or zip file
		dstFilename = strings.ReplaceAll(dstFilename, a.OS, "")
		dstFilename = strings.ReplaceAll(dstFilename, a.Arch, "")

		osData := osconfig.New(a.OS, a.Arch)
		for _, osAlias := range osData.GetAliases() {
			dstFilename = strings.ReplaceAll(dstFilename, osAlias, "")
		}
		for _, osArch := range osData.GetArchitectures() {
			dstFilename = strings.ReplaceAll(dstFilename, osArch, "")
		}

		dstFilename = strings.ReplaceAll(dstFilename, fmt.Sprintf("v%s", a.Version), "")
		dstFilename = strings.ReplaceAll(dstFilename, a.Version, "")

		dstFilename = versionReplace.ReplaceAllString(dstFilename, "")

		if a.OS == osconfig.Windows || strings.HasSuffix(dstFilename, ".exe") {
			dstFilename = strings.TrimSuffix(dstFilename, ".exe")
		}

		dstFilename = strings.TrimSpace(dstFilename)
		dstFilename = strings.TrimRight(dstFilename, "-")
		dstFilename = strings.TrimRight(dstFilename, "_")

		if a.OS == osconfig.Windows {
			dstFilename = fmt.Sprintf("%s.exe", dstFilename)
		}

		logrus.Tracef("post-dstFilename: %s", dstFilename)

		destBinaryName := dstFilename
		// Note: copy to the opt dir for organization
		destBinFilename := filepath.Join(optDir, destBinaryName)

		// Note: we put all symlinks into the bin dir
		defaultBinFilename := filepath.Join(binDir, dstFilename)

		versionedBinFilename := fmt.Sprintf("%s@%s", defaultBinFilename, strings.TrimLeft(a.Version, "v"))

		logrus.Debugf("copying executable: %s to %s", fullPath, destBinFilename)
		if err := a.copyFile(fullPath, destBinFilename); err != nil {
			return err
		}

		// create symlink
		// TODO: check if symlink exists
		// TODO: handle errors
		if runtime.GOOS == a.OS && runtime.GOARCH == a.Arch {
			logrus.Debugf("creating symlink: %s to %s", defaultBinFilename, destBinFilename)
			logrus.Debugf("creating symlink: %s to %s", versionedBinFilename, destBinFilename)
			_ = os.Remove(defaultBinFilename)
			_ = os.Remove(versionedBinFilename)
			_ = os.Symlink(destBinFilename, defaultBinFilename)
			_ = os.Symlink(destBinFilename, versionedBinFilename)
		}
	}

	if !found {
		return fmt.Errorf("the request binary was not found in the release")
	}

	return nil
}

func (a *Asset) Cleanup() error {
	if logrus.GetLevel() == logrus.TraceLevel {
		logrus.Tracef("walking tempdir")
		// walk the a.TempDir and log all the files
		err := filepath.Walk(a.TempDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			logrus.Tracef("file: %s", path)
			return nil
		})
		if err != nil {
			return err
		}
	}

	logrus.WithField("asset", a.GetName()).Tracef("cleaning up temp dir: %s", a.TempDir)
	return os.RemoveAll(a.TempDir)
}

func (a *Asset) Extract() error {
	var err error

	fileHandler, err := os.Open(a.DownloadPath)
	if err != nil {
		return err
	}

	a.TempDir, err = os.MkdirTemp("", common.NAME)
	if err != nil {
		return err
	}

	logrus.Debugf("opened and extracting file: %s", a.DownloadPath)

	return a.doExtract(fileHandler)
}

func (a *Asset) doExtract(in io.Reader) error {
	var buf bytes.Buffer
	tee := io.TeeReader(in, &buf)

	t, err := filetype.MatchReader(tee)
	if err != nil {
		return err
	}

	outputFile := io.MultiReader(&buf, in)

	logrus.Debugf("extracting file type: %s", t)

	var processor processorFunc

	switch t {
	case matchers.TypeTar:
		processor = a.processTar
	case matchers.TypeZip:
		processor = a.processZip
	case matchers.TypeBz2:
		processor = a.processBz2
	case matchers.TypeGz:
		processor = a.processGz
	case matchers.TypeXz:
		processor = a.processXz
	default:
		processor = a.processDirect
	}

	if processor != nil {
		newReader, err := processor(outputFile)
		if err != nil {
			return err
		}

		if newReader == nil {
			return nil
		}

		// In case of e.g. a .tar.gz, process the uncompressed archive by calling recursively
		return a.doExtract(newReader)
	}

	return nil
}

func (a *Asset) processDirect(in io.Reader) (io.Reader, error) {
	logrus.Tracef("processing direct file")
	outFile, err := os.Create(filepath.Join(a.TempDir, filepath.Base(a.DownloadPath)))
	if err != nil {
		return nil, err
	}

	if _, err := io.Copy(outFile, in); err != nil {
		return nil, err
	}

	a.Files = append(a.Files, &File{Name: filepath.Base(a.DownloadPath), Alias: a.GetName()})

	return nil, nil
}

func (a *Asset) processZip(in io.Reader) (io.Reader, error) {
	zr := zipstream.NewReader(in)
	a.Files = make([]*File, 0)

	for {
		header, err := zr.Next()

		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		target := filepath.Join(a.TempDir, header.Name)
		logrus.Tracef("zip > target %s", target)

		if header.Mode().IsDir() {
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return nil, err
				}
				logrus.Tracef("tar > create directory %s", target)
			}

			continue
		}

		// TODO(ek): do we need to somehow check the location in the zip file?
		// TODO(ek): should we cache the hashes of the files back to the main hash of the file?

		f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, header.Mode())
		if err != nil {
			return nil, err
		}

		// copy over contents
		if _, err := io.Copy(f, zr); err != nil {
			return nil, err
		}

		// manually close here after each file operation; deferring would cause each file close
		// to wait until all operations have completed.
		f.Close()

		a.Files = append(a.Files, &File{Name: header.Name})

		logrus.Tracef("zip > create file %s", target)
	}

	if len(a.Files) == 0 {
		return nil, fmt.Errorf("no files found in zip archive")
	}

	return nil, nil
}

func (a *Asset) processTar(in io.Reader) (io.Reader, error) {
	logrus.Trace("processing tar file")
	tr := tar.NewReader(in)
	a.Files = make([]*File, 0)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		// TODO(ek): do we need to somehow check the location in the tar file?

		target, err := sanitizeArchivePath(a.TempDir, header.Name)
		if err != nil {
			return nil, err
		}

		logrus.Tracef("tar > target %s", target)

		switch header.Typeflag {
		// if it's a dir, and it doesn't exist create it
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0755); err != nil {
					return nil, err
				}
				logrus.Tracef("tar > create directory %s", target)
			}
		// if it's a file create it
		case tar.TypeReg:
			baseDir := filepath.Dir(target)
			if _, err := os.Stat(baseDir); err != nil {
				if err := os.MkdirAll(baseDir, 0755); err != nil {
					return nil, err
				}
				logrus.Tracef("tar > create directory %s", baseDir)
			}

			convertedMode, err := int64ToUint32(header.Mode)
			if err != nil {
				return nil, err
			}

			f, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(convertedMode))
			if err != nil {
				return nil, err
			}

			// copy over contents
			if _, err := io.Copy(f, tr); err != nil { //nolint: gosec
				return nil, err
			}

			// manually close here after each file operation; deferring would cause each file close
			// to wait until all operations have completed.
			f.Close()

			a.Files = append(a.Files, &File{Name: header.Name})
			logrus.Tracef("tar > create file %s", target)
		}
	}

	if len(a.Files) == 0 {
		return nil, fmt.Errorf("no files in tar archive")
	}

	return nil, nil
}

func (a *Asset) processGz(in io.Reader) (io.Reader, error) {
	gr, err := gzip.NewReader(in)
	if err != nil {
		return nil, err
	}

	return gr, nil
}

func (a *Asset) processXz(in io.Reader) (io.Reader, error) {
	xr, err := xz.NewReader(in, 0)
	if err != nil {
		return nil, err
	}

	return xr, nil
}

func (a *Asset) processBz2(in io.Reader) (io.Reader, error) {
	br := bzip2.NewReader(in)
	return br, nil
}

func (a *Asset) GetGPGKeyID() (uint64, error) {
	if a.Type != Signature {
		return 0, fmt.Errorf("asset is not a signature: %s", a.GetName())
	}

	signatureContent, err := os.ReadFile(a.GetFilePath())
	if err != nil {
		return 0, fmt.Errorf("failed to read signature: %w", err)
	}

	// Parse the armored signature
	signature, err := crypto.NewPGPSignatureFromArmored(string(signatureContent))
	if err != nil {
		// Assume unarmored it not armored
		signature = crypto.NewPGPSignature(signatureContent)
	}

	ids, ok := signature.GetSignatureKeyIDs()
	if !ok {
		return 0, errors.New("signature does not contain a key ID")
	}

	return ids[0], nil
}

func int64ToUint32(value int64) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, errors.New("value out of range for uint32")
	}
	return uint32(value), nil
}

// sanitizeArchivePath ensures that the path is not tainted
// thanks https://github.com/securego/gosec/issues/324#issuecomment-935927967
func sanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}
