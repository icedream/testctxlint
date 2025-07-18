//go:build tools
// +build tools

// This file imports packages that are used when running go generate.
// This imports the packages so they are tracked in go.mod and won't be
// removed by go mod tidy.

package main

import (
	_ "github.com/josephspurrier/goversioninfo/cmd/goversioninfo"
)
