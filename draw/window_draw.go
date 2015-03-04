package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
)

type Color struct{ R, G, B, A float32 }

var Black = Color{0, 0, 0, 1}
var White = Color{1, 1, 1, 1}
var Gray = Color{0.5, 0.5, 0.5, 1}
var LightGray = Color{0.75, 0.75, 0.75, 1}
var DarkGray = Color{0.25, 0.25, 0.25, 1}
var Red = Color{1, 0, 0, 1}
var LightRed = Color{1, 0.5, 0.5, 1}
var DarkRed = Color{0.5, 0, 0, 1}
var Green = Color{0, 1, 0, 1}
var LightGreen = Color{0.5, 1, 0.5, 1}
var DarkGreen = Color{0, 0.5, 0, 1}
var Blue = Color{0, 0, 1, 1}
var LightBlue = Color{0.5, 0.5, 1, 1}
var DarkBlue = Color{0, 0, 0.5, 1}
var Purple = Color{1, 0, 1, 1}
var LightPurple = Color{1, 0.5, 1, 1}
var DarkPurple = Color{0.5, 0, 0.5, 1}
var Yellow = Color{1, 1, 0, 1}
var LightYellow = Color{1, 1, 0.5, 1}
var DarkYellow = Color{0.5, 0.5, 0, 1}
var Cyan = Color{0, 1, 1, 1}
var LightCyan = Color{0.5, 1, 1, 1}
var DarkCyan = Color{0, 0.5, 0.5, 1}

func (w *Window) DrawEllipse(x, y, width, height int, color Color) {
	if width == 1 {
		w.DrawLine(x, y, x, y+height-1, color)
		return
	}
	if height == 1 {
		w.DrawLine(x, y, x+width-1, y, color)
		return
	}

	points := ellipsePoints(x, y, width, height)
	w.setColor(color)
	for _, p := range points {
		w.DrawPoint(p.x, p.y, color)
	}
}

func ellipsePoints(left, top, width, height int) []point {
	if width == 0 || height == 0 {
		return nil
	}
	if width == 1 && height == 1 {
		return []point{{left, top}, {left, top}}
	}

	centerX := left + width/2
	centerY := top + height/2
	points := make([]point, 0, height*2)
	addPoint := func(x, y int) {
		points = append(points, point{centerX + x, centerY + y})
		points = append(points, point{centerX + x, centerY - y})
		points = append(points, point{centerX - x, centerY + y})
		points = append(points, point{centerX - x, centerY - y})
	}

	xRadius := width / 2
	yRadius := height / 2
	a2 := xRadius * xRadius
	b2 := yRadius * yRadius
	fa2 := 4 * a2
	fb2 := 4 * b2

	for x, y, sigma := 0, yRadius, 2*b2+a2*(1-2*yRadius); b2*x <= a2*y; x++ {
		addPoint(x, y)
		if sigma >= 0 {
			sigma += fa2 * (1 - y)
			y--
		}
		sigma += b2 * ((4 * x) + 6)
	}

	for x, y, sigma := xRadius, 0, 2*a2+b2*(1-2*xRadius); a2*y <= b2*x; y++ {
		addPoint(x, y)
		if sigma >= 0 {
			sigma += fb2 * (1 - x)
			x--
		}
		sigma += a2 * ((4 * y) + 6)
	}

	return points
}

type point struct{ x, y int }

func (w *Window) DrawPoint(x, y int, color Color) {
	w.setColor(color)
	w.renderer.DrawPoint(x, y)
}

func (w *Window) setColor(color Color) {
	w.renderer.SetDrawColor(
		channel(color.R),
		channel(color.G),
		channel(color.B),
		channel(color.A))
}

func channel(value float32) uint8 {
	if value < 0 {
		value = 0
	}
	if value > 1 {
		value = 1
	}
	return uint8(value * 255)
}

func (w *Window) DrawLine(fromX, fromY, toX, toY int, color Color) {
	w.setColor(color)
	w.line(fromX, fromY, toX, toY)
}

func (w *Window) line(fromX, fromY, toX, toY int) {
	w.renderer.DrawLine(fromX, fromY, toX, toY)
}

func (w *Window) DrawRect(x, y, width, height int, color Color) {
	if width == 0 || height == 0 {
		return
	}
	w.setColor(color)
	x2 := x + width - 1
	y2 := y + height - 1
	w.line(x, y, x2, y)
	w.line(x2, y, x2, y2)
	w.line(x2, y2, x, y2)
	w.line(x, y2, x, y)
}

func (w *Window) FillRect(x, y, width, height int, color Color) {
	if width == 0 || height == 0 {
		return
	}
	w.setColor(color)
	w.renderer.FillRect(&sdl.Rect{int32(x), int32(y), int32(width), int32(height)})
}

func (w *Window) DrawImageFile(path string, x, y int) error {
	w.loadImageIfNecessary(path)
	img := w.textures[path]
	if img == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	_, _, width, height, _ := img.Query()
	w.renderer.Copy(
		img,
		&sdl.Rect{0, 0, int32(width), int32(height)},
		&sdl.Rect{int32(x), int32(y), int32(width), int32(height)})
	return nil
}

func (w *Window) loadImageIfNecessary(path string) {
	if _, ok := w.textures[path]; ok {
		return
	}
	img, err := img.Load(path)
	if err != nil {
		w.textures[path] = nil
		return
	}
	defer img.Free()
	texture, err := w.renderer.CreateTextureFromSurface(img)
	if err != nil {
		w.textures[path] = nil
		return
	}
	w.textures[path] = texture
}

func (w *Window) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1.0, color)
}

func (w *Window) DrawScaledText(text string, x, y int, scale float32, color Color) {
	w.fontTexture.SetColorMod(channel(color.R), channel(color.G), channel(color.B))
	_, _, width, height, _ := w.fontTexture.Query()
	width /= 16
	height /= 16
	src := sdl.Rect{0, 0, int32(width), int32(height)}
	dest := src
	dest.W = int32(float32(dest.W) * scale)
	dest.H = int32(float32(dest.H) * scale)
	dest.X = int32(x)
	dest.Y = int32(y)
	for _, char := range []byte(text) {
		if char == '\n' {
			dest.X = int32(x)
			dest.Y += dest.H
			continue
		}
		src.X = int32((int(char) % 16) * width)
		src.Y = int32((int(char) / 16) * height)
		w.renderer.Copy(w.fontTexture, &src, &dest)
		dest.X += dest.W
	}
}
