package main

// This file contains go:generate commands for building the Windows metadata only.
// This is separate from icon generation to avoid requiring ImageMagick in CI.
// Run `go generate` with this file to regenerate only the metadata resources.

//go:generate go run gen_version.go
