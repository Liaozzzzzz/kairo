package main

import (
	"embed"
	"io/fs"
)

//go:embed assets/bin/*
var embeddedBin embed.FS

func readEmbedded(name string) ([]byte, error) {
	return fs.ReadFile(embeddedBin, "assets/bin/"+name)
}
