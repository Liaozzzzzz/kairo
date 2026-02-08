//go:build shared

package main

import "errors"

func readEmbedded(name string) ([]byte, error) {
	return nil, errors.New("assets not embedded in shared mode")
}
