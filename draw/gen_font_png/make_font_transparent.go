//go:build exclude

package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

func main() {
	if err := run(); err != nil {
		panic(err)
	}
}

func run() error {
	img, err := loadImage("font.png")
	if err != nil {
		return err
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	output := image.NewNRGBA(b)

	for y := range h {
		for x := range w {
			gray, _, _, _ := img.At(x, y).RGBA()
			alpha := uint8(gray >> 8)
			output.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: alpha})
		}
	}

	return saveImage(output, "../font.png")
}

func loadImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return png.Decode(f)
}

func saveImage(img image.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}
