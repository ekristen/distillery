package config

// Settings - settings to control the behavior of distillery
type Settings struct {
	// ChecksumMissing - behavior when a checksum file is missing, this defaults to "warn", other options are "error" and "ignore"
	ChecksumMissing string `yaml:"checksum-missing" toml:"checksum-missing"`
	// ChecksumMismatch - behavior when a checksum file is missing, this defaults to "warn", other options are "error" and "ignore"
	SignatureMissing string `yaml:"signature-missing" toml:"signature-missing"`

	/// ChecksumUnknown - behavior when a checksum method cannot be determined, this defaults to "warn", other options are "error" and "ignore"
	ChecksumUnknown string `yaml:"checksum-unknown" toml:"checksum-unknown"`
}

// Defaults - set the default values for the settings
func (s *Settings) Defaults() {
	if s.ChecksumMissing == "" {
		s.ChecksumMissing = "warn"
	}

	if s.SignatureMissing == "" {
		s.SignatureMissing = "warn"
	}

	if s.ChecksumUnknown == "" {
		s.ChecksumUnknown = "warn"
	}
}
