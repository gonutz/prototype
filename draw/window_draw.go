package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
	"math"
	"strings"
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
var Brown = Color{0.5, 0.2, 0, 1}
var LightBrown = Color{0.75, 0.3, 0, 1}

func (w *Window) DrawEllipse(x, y, width, height int, color Color) {
	points := ellipsePoints(x, y, width, height)
	if len(points) > 0 {
		w.setColor(color)
		w.renderer.DrawPoints(makeSDLpoints(points))
	}
}

func (w *Window) FillEllipse(x, y, width, height int, color Color) {
	points := ellipsePoints(x, y, width, height)
	if len(points) > 0 {
		w.setColor(color)
		w.renderer.DrawLines(makeSDLpoints(points))
	}
}

func makeSDLpoints(from []point) []sdl.Point {
	p := make([]sdl.Point, len(from))
	for i, in := range from {
		p[i].X = int32(in.x)
		p[i].Y = int32(in.y)
	}
	return p
}

func ellipsePoints(left, top, width, height int) []point {
	if width <= 0 || height <= 0 {
		return nil
	}

	if height > width {
		return flipPoints(ellipsePoints(top, left, height, width))
	}

	var points []point
	a := float64(width) / 2.0
	b := float64(height) / 2.0
	bSquare := b * b
	bSquareOverASquare := bSquare / (a * a)
	yOf := func(x float64) float64 {
		square := bSquare - x*x*bSquareOverASquare
		if square <= 0.0 {
			return 0.0
		}
		return math.Sqrt(square)
	}
	round := func(x float64) int { return int(x + 0.49) }
	startX := 0.0
	if width%2 == 0 {
		startX = 0.5
	}
	endX := a + 1.1
	lastY := round(yOf(startX))
	for x := startX; x < endX; x += 1.0 {
		ix := int(x)
		iy := round(yOf(x))
		for y := lastY; y != iy; y-- {
			points = append(points, point{ix, y})
		}
		points = append(points, point{ix, iy})
		lastY = iy
	}
	all := make([]point, len(points)*4)
	for i, p := range points {
		all[i*4+0] = point{p.x + left + width/2, p.y + top + height/2}
		all[i*4+1] = point{-p.x + left + width/2, p.y + top + height/2}
		all[i*4+2] = point{p.x + left + width/2, -p.y + top + height/2}
		all[i*4+3] = point{-p.x + left + width/2, -p.y + top + height/2}
	}
	return all
}

func flipPoints(points []point) []point {
	for i, p := range points {
		points[i].x, points[i].y = p.y, p.x
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
	if width <= 0 || height <= 0 {
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
	if width <= 0 || height <= 0 {
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

func (win *Window) DrawImageFileTo(path string, x, y, w, h, degrees int) error {
	win.loadImageIfNecessary(path)
	img := win.textures[path]
	if img == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	win.renderer.CopyEx(
		img,
		nil,
		&sdl.Rect{int32(x), int32(y), int32(w), int32(h)},
		float64(degrees),
		nil,
		sdl.FLIP_NONE)
	return nil
}

func (w *Window) DrawImageFilePortion(path string, srcX, srcY, srcW, srcH, toX, toY int) error {
	w.loadImageIfNecessary(path)
	img := w.textures[path]
	if img == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	w.renderer.Copy(
		img,
		&sdl.Rect{int32(srcX), int32(srcY), int32(srcW), int32(srcH)},
		&sdl.Rect{int32(toX), int32(toY), int32(srcW), int32(srcH)})
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

func (win *Window) GetTextSize(text string) (w, h int) {
	return win.GetScaledTextSize(text, 1.0)
}

func (win *Window) GetScaledTextSize(text string, scale float32) (w, h int) {
	sdl.SetHint(sdl.HINT_RENDER_SCALE_QUALITY, "linear")
	if len(text) == 0 {
		return 0, 0
	}
	_, _, width, height, _ := win.fontTexture.Query()
	width /= 16
	height /= 16
	w = int(float32(width) * scale)
	h = int(float32(height) * scale)
	lines := strings.Split(text, "\n")
	maxLineW := 0
	for _, line := range lines {
		if len(line) > maxLineW {
			maxLineW = len(line)
		}
	}
	return w * maxLineW, h * len(lines)
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
		src.X = int32((int(char) % 16)) * width
		src.Y = int32((int(char) / 16)) * height
		w.renderer.Copy(w.fontTexture, &src, &dest)
		dest.X += dest.W
	}
}
