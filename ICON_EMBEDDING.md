# Application Icon and Metadata Embedding

This document describes how the application icon and metadata are embedded into Windows executables.

## Overview

The testctxlint Windows executables include an embedded application icon based on the project logo, along with proper application metadata (title, author, version information). This provides a better user experience when the executable is displayed in file explorers or task managers.

## Files

- `assets/logo-icon.svg` - Original SVG logo
- `assets/logo-icon.ico` - Windows ICO format icon (generated from SVG)
- `cmd/testctxlint/icon_windows_amd64.syso` - Windows resource file (compiled from ICO and metadata)
- `cmd/testctxlint/gen_icon.go` - Go generate script for building icon resources
- `cmd/testctxlint/gen_version.go` - Script to generate Windows resource file with dynamic version
- `cmd/testctxlint/versioninfo.json` - Base configuration for Windows resource metadata

## How It Works

1. The SVG logo is converted to ICO format with multiple sizes (16x16, 32x32, 48x48, 64x64, 128x128, 256x256)
2. The version is dynamically determined from Git tags (or defaults to "0.0.0.0" for development)
3. The ICO file and metadata are compiled into a Windows resource file (.syso) using the `goversioninfo` tool
4. The .syso file is automatically included in Windows builds by the Go compiler
5. The .syso file is ignored on non-Windows builds due to the `_windows_amd64` suffix

## Embedded Metadata

The Windows executable includes the following metadata:
- **Application Title**: testctxlint
- **Author/Company**: Carl Kittelberger
- **Description**: A linter for Go that identifies context.Context usage in testing
- **Version**: Dynamically determined from Git tags (e.g., "v1.2.3" becomes "1.2.3.0")
- **Copyright**: Copyright (c) 2024 Carl Kittelberger

## Regenerating the Icon and Metadata

If the logo changes or metadata needs to be updated, you can regenerate the embedded resources using `go generate`:

```bash
cd cmd/testctxlint
go generate
```

This will:
1. Convert the SVG logo to ICO format (requires ImageMagick)
2. Generate the Windows resource file with the current version from Git tags and all metadata using the `goversioninfo` tool

### Manual Regeneration

If you prefer to run the commands manually:

```bash
# Convert SVG to ICO (requires ImageMagick)
magick convert -background none assets/logo-icon.svg -define icon:auto-resize=256,128,64,48,32,16 assets/logo-icon.ico

# Generate Windows resource file with dynamic version and metadata
cd cmd/testctxlint
go run gen_version.go
```

### Updating Metadata

To update the application metadata (title, author, version, etc.), edit the `cmd/testctxlint/gen_version.go` file to modify the command-line arguments passed to `goversioninfo`, or edit the `cmd/testctxlint/versioninfo.json` file for more complex changes. Then run `go generate` to regenerate the resource file.

The version is automatically determined from Git tags in the format "v1.2.3" (converted to "1.2.3.0" for Windows). For development builds without tags, the version defaults to "0.0.0.0".

## Verification

To verify the icon and metadata are embedded:

1. Build for Windows: `GOOS=windows GOARCH=amd64 go build -o testctxlint.exe ./cmd/testctxlint`
2. The executable should be ~200KB larger than without the icon
3. On Windows, the executable should display the embedded icon in file explorers
4. Right-click the executable and select "Properties" → "Details" to see the embedded metadata