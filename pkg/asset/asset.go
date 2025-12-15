package asset

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"

	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/gabriel-vasile/mimetype"
	"github.com/h2non/filetype"
	"github.com/h2non/filetype/matchers"
	"github.com/mholt/archives"
	"github.com/sirupsen/logrus"

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

func (a *Asset) GetBaseName() string {
	filename := a.GetName()
	for {
		newFilename := filename
		newExt := filepath.Ext(newFilename)
		if len(newExt) > 5 || strings.Contains(newExt, "_") {
			break
		}

		newFilename = strings.TrimSuffix(newFilename, newExt)

		if newFilename == filename {
			break
		}

		filename = newFilename
	}

	return filename
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

		if !file.Installable && a.OS == osconfig.Linux && m.String() == "application/x-sharedlib" {
			file.Installable = a.determineELF(fullPath)
		}
	}
}

// determineELF determines if the file is an ELF binary, this is a fallback for linux should the mimetype be unable
// to determine if the file is an executable.
// Note: this is special code to check if the file is an ELF binary because sometimes the mimetype on linux due to the
// fact that depending on how gcc is configured the mimetype might be detected as application/x-sharedlib instead
func (a *Asset) determineELF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		logrus.WithError(err).Tracef("unable to open file for elf determination1: %s", path)
		return false
	}
	defer f.Close()

	magic := make([]byte, 4)
	_, err = f.Read(magic)
	if err != nil {
		logrus.WithError(err).Trace("error reading file")
		return false
	}

	elfMagic := []byte{0x7F, 'E', 'L', 'F'}
	return bytes.Equal(magic, elfMagic)
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

func (a *Asset) doExtract(stream io.Reader) error {
	logrus.Debug("identifying archive format")
	format, stream, err := archives.Identify(context.TODO(), a.Extension, stream)
	if err != nil && !errors.Is(err, archives.NoMatch) {
		return err
	}

	if errors.Is(err, archives.NoMatch) && a.GetType() == Archive {
		logrus.Warn("unable to identify archive format")
		return errors.New("unable to identify or invalid archive format")
	}

	logrus.Debug("identified archive format: ", format)

	if ex, ok := format.(archives.Extractor); ok {
		logrus.Debug("extracting archive")
		if err := ex.Extract(context.TODO(), stream, a.processArchive); err != nil {
			return err
		}
	} else {
		logrus.Debug("processing direct file")
		if err := a.processDirect(stream); err != nil {
			return err
		}
	}

	return nil
}

func (a *Asset) processDirect(in io.Reader) error {
	logrus.Tracef("processing direct file")
	outFile, err := os.Create(filepath.Join(a.TempDir, filepath.Base(a.DownloadPath)))
	if err != nil {
		return err
	}

	if _, err := io.Copy(outFile, in); err != nil {
		return err
	}

	a.Files = append(a.Files, &File{Name: filepath.Base(a.DownloadPath), Alias: a.GetName()})

	return nil
}

func (a *Asset) processArchive(ctx context.Context, f archives.FileInfo) error {
	// do something with the file here; or, if you only want a specific file or directory,
	// just return until you come across the desired f.NameInArchive value(s)
	target := filepath.Join(a.TempDir, f.Name())
	logrus.Tracef("zip > target %s", target)

	if f.Mode().IsDir() {
		if _, err := os.Stat(target); err != nil {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
			logrus.Tracef("tar > create directory %s", target)
		}

		return nil
	}

	tc, err := f.Open()
	if err != nil {
		return err
	}

	nf, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, f.Mode())
	if err != nil {
		return err
	}

	// copy over contents
	if _, err := io.Copy(nf, tc); err != nil {
		return err
	}

	a.Files = append(a.Files, &File{Name: f.Name()})

	logrus.Tracef("zip > create file %s", target)

	return nil
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
