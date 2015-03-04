package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
)

type Color struct{ R, G, B, A float32 }

var Black = Color{0, 0, 0, 1}
var White = Color{1, 1, 1, 1}
var Red = Color{1, 0, 0, 1}
var Green = Color{0, 1, 0, 1}
var Blue = Color{0, 0, 1, 1}
var Purple = Color{1, 0, 1, 1}
var Yellow = Color{1, 1, 0, 1}
var Cyan = Color{0, 1, 1, 1}

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

	points := make([]point, 0, height*2)
	centerX := left + width/2
	centerY := top + height/2
	xRadius := width / 2
	yRadius := height / 2
	a2 := 2 * xRadius * xRadius
	b2 := 2 * yRadius * yRadius
	x := xRadius
	y := 0
	addPoint := func() {
		points = append(points, point{centerX + x, centerY + y})
		points = append(points, point{centerX + x, centerY - y})
		points = append(points, point{centerX - x, centerY + y})
		points = append(points, point{centerX - x, centerY - y})
	}
	xChange := yRadius * yRadius * (1 - 2*xRadius)
	yChange := xRadius * xRadius
	ellipseError := 0
	stoppingX := b2 * xRadius
	stoppingY := 0
	for stoppingX >= stoppingY {
		addPoint()
		y++
		stoppingY += a2
		ellipseError += yChange
		yChange += a2
		if 2*ellipseError+xChange > 0 {
			x--
			stoppingX -= b2
			ellipseError += xChange
			xChange += b2
		}
	}

	x = 0
	y = yRadius
	xChange = yRadius * yRadius
	yChange = xRadius * xRadius * (1 - 2*yRadius)
	ellipseError = 0
	stoppingX = 0
	stoppingY = a2 * yRadius
	for stoppingX <= stoppingY {
		addPoint()
		x++
		stoppingX += b2
		ellipseError += xChange
		xChange += b2
		if 2*ellipseError+yChange > 0 {
			y--
			stoppingY -= a2
			ellipseError += yChange
			yChange += a2
		}
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
