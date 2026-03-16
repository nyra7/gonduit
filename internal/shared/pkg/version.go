package pkg

var (
	AppName   = "gonduit"
	Version   = "dev"
	Commit    = "init"
	BuildDate = "unknown"
)

// IsDebug returns true if the build is a debug build
// The Makefile overrides the version value when building a release binary
func IsDebug() bool {
	return Version == "dev"
}
