package kit

// Version is the version of Go-Kit to generate templates for.
var Version = "bbb2306"

// VersionsSupported are the current versions of Go-Kit that truss can
// generate files for.
var VersionsSupported = []string{
	"bbb2306",
}

// VersionNotSupported validates user input for go-kit supported versions
func VersionNotSupported(reqVersion string) bool {
	for _, supportedVersion := range VersionsSupported {
		if supportedVersion == reqVersion {
			return false
		}
	}
	return true
}
