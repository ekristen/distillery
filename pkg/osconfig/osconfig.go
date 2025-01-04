package osconfig

const (
	Windows = "windows"
	Linux   = "linux"
	Darwin  = "darwin"
	FreeBSD = "freebsd"

	AMD64 = "amd64"
	ARM64 = "arm64"
	ARM32 = "arm32" // note: there is no such thing as arm32, but this is just to standardize the naming
	AMD32 = "amd32" // note: there is no such thing as amd32, but this is just to standardize the naming
)

var (
	AMD64Architectures = []string{"amd64", "x86_64", "x86-64", "64bit", "x64", "64-bit"}
	ARM64Architectures = []string{"arm64", "aarch64", "armv8-a", "arm64-bit"}
	X86Architectures   = []string{"x86", "i686", "i386"}
	ARM32Architectures = []string{"armv7", "armv6", "armv5", "armv4"}
)

type OS struct {
	Name          string
	Arch          string
	Aliases       []string
	Architectures []string
	Extensions    []string
}

func (o *OS) GetOS() []string {
	return append([]string{o.Name}, o.Aliases...)
}

func (o *OS) GetAliases() []string {
	return o.Aliases
}

func (o *OS) GetArchitecture() string {
	return o.Arch
}

func (o *OS) GetArchitectures() []string {
	return o.Architectures
}

func (o *OS) GetExtensions() []string {
	return o.Extensions
}

func (o *OS) InvalidOS() []string {
	switch o.Name {
	case Windows:
		return []string{Linux, Darwin, FreeBSD}
	case Linux:
		return []string{Windows, Darwin}
	case Darwin:
		return []string{Windows, Linux, FreeBSD}
	}

	return []string{}
}

func (o *OS) InvalidArchitectures() []string {
	switch o.Arch {
	case ARM64:
		return AMD64Architectures
	case AMD64:
		return ARM64Architectures
	case ARM32:
		return ARM32Architectures
	case AMD32:
		return ARM64Architectures
	}

	return []string{}
}

func New(os, arch string) *OS {
	newOS := &OS{
		Name:          os,
		Arch:          arch,
		Architectures: []string{arch},
	}

	switch os {
	case Windows:
		newOS.Aliases = []string{"win"}
		newOS.Extensions = []string{".exe"}
	case Linux:
		newOS.Aliases = []string{}
		newOS.Extensions = []string{".AppImage"}
	case Darwin:
		newOS.Aliases = []string{"osx", "macos", "mac", "apple", "ventura", "sonoma", "sequoia"}
		newOS.Architectures = append(newOS.Architectures, "universal")
	}

	switch arch {
	case AMD64:
		newOS.Architectures = append(newOS.Architectures, AMD64Architectures...)
	case ARM64:
		newOS.Architectures = append(newOS.Architectures, ARM64Architectures...)
	case ARM32:
		newOS.Architectures = append(newOS.Architectures, ARM32Architectures...)
	case AMD32:
		newOS.Architectures = append(newOS.Architectures, X86Architectures...)
	}

	newOS.Architectures = removeDuplicateStr(newOS.Architectures)

	return newOS
}

func removeDuplicateStr(strSlice []string) []string {
	allKeys := make(map[string]bool)
	var list []string
	for _, item := range strSlice {
		if _, value := allKeys[item]; !value {
			allKeys[item] = true
			list = append(list, item)
		}
	}
	return list
}
