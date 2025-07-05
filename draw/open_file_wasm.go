//go:build js && wasm
// +build js,wasm

package draw

import "io"

// DefaultOpenFile for WASM builds is nil so that the WASM port knows to load
// from URL.
var DefaultOpenFile func(path string) (io.ReadCloser, error) = nil
