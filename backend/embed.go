package main

import "embed"

//go:embed all:dist
var embeddedFrontend embed.FS
