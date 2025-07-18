package main

// This file contains go:generate commands for building the Windows application icon only.
// Run `go generate` with this file to regenerate the icon file.
// For CI/release purposes, use gen_syso.go instead to generate only the metadata.
//
// Prerequisites:
// - ImageMagick (for magick convert command)

//go:generate magick convert -background none ../../assets/logo-icon.svg -define icon:auto-resize=256,128,64,48,32,16 ../../assets/logo-icon.ico
