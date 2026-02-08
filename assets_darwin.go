//go:build darwin && !shared

package main

import "embed"

//go:embed assets/bin/darwin/*
var embeddedAssets embed.FS

func readEmbedded(name string) ([]byte, error) {
	return embeddedAssets.ReadFile("assets/bin/darwin/" + name)
}
