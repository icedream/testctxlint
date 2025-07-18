//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// getVersion attempts to get the version from git tags or defaults to "0.0.0.0"
func getVersion() string {
	// Try to get version from git describe
	cmd := exec.Command("git", "describe", "--tags", "--abbrev=0")
	output, err := cmd.Output()
	if err == nil {
		version := strings.TrimSpace(string(output))
		if version != "" {
			return strings.TrimPrefix(version, "v")
		}
	}

	// Try to get version from git tag --points-at HEAD
	cmd = exec.Command("git", "tag", "--points-at", "HEAD")
	output, err = cmd.Output()
	if err == nil {
		tags := strings.Fields(string(output))
		if len(tags) > 0 {
			return strings.TrimPrefix(tags[0], "v")
		}
	}

	// Default to dev version
	return "0.0.0.0"
}

// parseVersion parses a version string into major, minor, patch, build components
func parseVersion(version string) (major, minor, patch, build int) {
	// Match semantic version pattern
	re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(?:\.(\d+))?(?:-.*)?$`)
	matches := re.FindStringSubmatch(version)
	
	if len(matches) >= 4 {
		fmt.Sscanf(matches[1], "%d", &major)
		fmt.Sscanf(matches[2], "%d", &minor)
		fmt.Sscanf(matches[3], "%d", &patch)
		if len(matches) >= 5 && matches[4] != "" {
			fmt.Sscanf(matches[4], "%d", &build)
		}
		return
	}
	
	// If not a standard semantic version, try to parse as major.minor
	re = regexp.MustCompile(`^(\d+)\.(\d+)(?:-.*)?$`)
	matches = re.FindStringSubmatch(version)
	
	if len(matches) >= 3 {
		fmt.Sscanf(matches[1], "%d", &major)
		fmt.Sscanf(matches[2], "%d", &minor)
		return
	}
	
	return 0, 0, 0, 0
}

func main() {
	version := getVersion()
	major, minor, patch, build := parseVersion(version)
	
	fmt.Fprintf(os.Stderr, "Generating Windows resource file with version: %s (%d.%d.%d.%d)\n", version, major, minor, patch, build)
	
	// Generate the resource file using goversioninfo
	cmd := exec.Command("go", "run", "-mod=mod", "github.com/josephspurrier/goversioninfo/cmd/goversioninfo",
		"-64", // Generate 64-bit binaries
		"-o", "icon_windows_amd64.syso",
		"-icon", "../../assets/logo-icon.ico",
		"-company", "Carl Kittelberger",
		"-product-name", "testctxlint",
		"-file-version", version,
		"-product-version", version,
		"-copyright", "Copyright (c) 2025 Carl Kittelberger",
		"-description", "testctxlint",
		"-internal-name", "testctxlint",
		"-original-name", "testctxlint.exe",
		"-comment", "A linter for Go that identifies context.Context usage in testing",
		fmt.Sprintf("-ver-major=%d", major),
		fmt.Sprintf("-ver-minor=%d", minor),
		fmt.Sprintf("-ver-patch=%d", patch),
		fmt.Sprintf("-ver-build=%d", build),
		fmt.Sprintf("-product-ver-major=%d", major),
		fmt.Sprintf("-product-ver-minor=%d", minor),
		fmt.Sprintf("-product-ver-patch=%d", patch),
		fmt.Sprintf("-product-ver-build=%d", build),
		"versioninfo.json")
	
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error running goversioninfo: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Fprintln(os.Stderr, "Successfully generated Windows resource file")
}