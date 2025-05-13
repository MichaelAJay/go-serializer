package serializer

// Version information
const (
	// Version is the current version of the package
	Version = "v0.1.0"

	// VersionMajor is the major version number
	VersionMajor = 0

	// VersionMinor is the minor version number
	VersionMinor = 1

	// VersionPatch is the patch version number
	VersionPatch = 0
)

// VersionString returns the full version string
func VersionString() string {
	return Version
}

// VersionInfo returns the version information as a map
func VersionInfo() map[string]int {
	return map[string]int{
		"major": VersionMajor,
		"minor": VersionMinor,
		"patch": VersionPatch,
	}
}
