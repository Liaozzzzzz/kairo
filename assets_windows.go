//go:build windows && !shared

package main

import "embed"

//go:embed assets/bin/windows/*
var embeddedAssets embed.FS

func readEmbedded(name string) ([]byte, error) {
	return embeddedAssets.ReadFile("assets/bin/windows/" + name)
}
