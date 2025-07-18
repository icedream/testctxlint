package main

// This file contains go:generate commands for building the Windows application icon with metadata.
// Run `go generate` in this directory to regenerate the icon resources.
//
// Prerequisites:
// - ImageMagick (for magick convert command)
// - goversioninfo tool (automatically downloaded via go run)

//go:generate magick convert -background none ../../assets/logo-icon.svg -define icon:auto-resize=256,128,64,48,32,16 ../../assets/logo-icon.ico
//go:generate go run gen_version.go
