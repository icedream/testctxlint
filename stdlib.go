package testctxlint

import "regexp"

// Regexp for matching go tags. The groups are:
// 1  the major.minor version
// 2  the patch version, or empty if none
// 3  the entire prerelease, if present
// 4  the prerelease type ("beta" or "rc")
// 5  the prerelease number
//
// Copied from golang.org/x/pkgsite/internal/stdlib.
var tagRegexp = regexp.MustCompile(`^go(\d+\.\d+)(\.\d+|)((beta|rc)(\d+))?$`)

// VersionForTag returns the semantic version for the Go tag, or "" if
// tag doesn't correspond to a Go release or beta tag. In special cases,
// when the tag specified is either `latest` or `master` it will return the tag.
// Examples:
//
//	"go1" => "v1.0.0"
//	"go1.2" => "v1.2.0"
//	"go1.13beta1" => "v1.13.0-beta.1"
//	"go1.9rc2" => "v1.9.0-rc.2"
//	"latest" => "latest"
//	"master" => "master"
//
// Copied from golang.org/x/pkgsite/internal/stdlib.
func versionFromGoVersion(tag string) string {
	// Special cases for go1.
	if tag == "go1" {
		return "v1.0.0"
	}
	if tag == "go1.0" {
		return ""
	}
	// removed support for latest/master
	m := tagRegexp.FindStringSubmatch(tag)
	if m == nil {
		return ""
	}
	version := "v" + m[1]
	if m[2] != "" {
		version += m[2]
	} else {
		version += ".0"
	}
	if m[3] != "" {
		version += "-" + m[4] + "." + m[5]
	}
	return version
}
