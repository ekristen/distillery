package common

// NAME of the App
var NAME = "distillery"

// SUMMARY of the Version
var SUMMARY = "v1.0.0"

// BRANCH of the Version
var BRANCH = "dev"

// VERSION of Release
var VERSION = "1.0.0"

var COMMIT = "dirty"

// AppVersion --
var AppVersion AppVersionInfo

// AppVersionInfo --
type AppVersionInfo struct {
	Name    string
	Version string
	Branch  string
	Summary string
	Commit  string
}

func init() {
	AppVersion = AppVersionInfo{
		Name:    NAME,
		Version: VERSION,
		Branch:  BRANCH,
		Summary: SUMMARY,
		Commit:  COMMIT,
	}
}
