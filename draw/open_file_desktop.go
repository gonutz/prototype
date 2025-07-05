//go:build !js
// +build !js

package draw

import (
	"io"
	"os"
)

// DefaultOpenFile on desktop loads the file from disk.
var DefaultOpenFile = func(path string) (io.ReadCloser, error) {
	return os.Open(path)
}
