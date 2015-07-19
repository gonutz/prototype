// +build sdl2

//!glfw

package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
	"github.com/veandco/go-sdl2/sdl_mixer"
	"math"
	"strings"
	"sync"
	"time"
	"unsafe"
)

type window struct {
	Events     []sdl.Event
	MouseMoved bool
	Mouse      struct{ X, Y int }

	update      UpdateFunction
	window      *sdl.Window
	running     bool
	renderer    *sdl.Renderer
	textures    map[string]*sdl.Texture
	soundChunks map[string]*mix.Chunk
	fontTexture *sdl.Texture
	keyDown     map[string]bool
	mouseDown   map[MouseButton]bool
	clicks      []MouseClick
}

var windowRunningMutex sync.Mutex

func RunWindow(title string, width, height int, flags int, update UpdateFunction) error {
	windowRunningMutex.Lock()

	if update == nil {
		return errors.New("Update function was nil.")
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return err
	}
	defer sdl.Quit()

	var sdlFlags uint32
	if flags&Resizable > 0 {
		sdlFlags |= sdl.WINDOW_RESIZABLE
	}
	sdlWindow, renderer, err := sdl.CreateWindowAndRenderer(width, height, sdlFlags)
	if err != nil {
		return err
	}
	defer sdlWindow.Destroy()
	defer renderer.Destroy()
	sdlWindow.SetTitle(title)
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	if err := mix.OpenAudio(44100, mix.DEFAULT_FORMAT, 1, 512); err != nil {
		return err
	}
	defer mix.CloseAudio()

	win := &window{
		running:     true,
		update:      update,
		window:      sdlWindow,
		renderer:    renderer,
		textures:    make(map[string]*sdl.Texture),
		soundChunks: make(map[string]*mix.Chunk),
		keyDown:     make(map[string]bool),
		mouseDown:   make(map[MouseButton]bool),
	}
	win.createBitmapFont()
	win.runMainLoop()
	win.close()

	windowRunningMutex.Unlock()
	return nil
}

func (w *window) createBitmapFont() {
	ptr := unsafe.Pointer(&bitmapFontWhitePng[0])
	rwops := sdl.RWFromMem(ptr, len(bitmapFontWhitePng))
	texture, err := img.LoadTexture_RW(w.renderer, rwops, 0)
	if err != nil {
		panic(err)
	}
	w.fontTexture = texture
}

func (w *window) runMainLoop() {
	w.renderer.SetDrawColor(0, 0, 0, 0)
	lastUpdateTime := time.Now().Add(-time.Hour)
	const updateInterval = 1.0 / 60.0
	for w.running {
		for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
			switch event := e.(type) {
			case *sdl.QuitEvent:
				w.running = false
			case *sdl.MouseMotionEvent:
				w.Mouse.X = int(event.X)
				w.Mouse.Y = int(event.Y)
				w.MouseMoved = true
			case *sdl.MouseButtonEvent:
				if event.State == sdl.PRESSED {
					w.clicks = append(w.clicks, makeClick(event))
					w.mouseDown[MouseButton(event.Button)] = true
				}
				if event.State == sdl.RELEASED {
					w.mouseDown[MouseButton(event.Button)] = false
				}
			case *sdl.KeyDownEvent:
				w.setKeyDown(event.Keysym.Sym, true)
			case *sdl.KeyUpEvent:
				w.setKeyDown(event.Keysym.Sym, false)
			}
			w.Events = append(w.Events, e)
		}

		now := time.Now()
		if now.Sub(lastUpdateTime).Seconds() > updateInterval {
			w.renderer.SetDrawColor(0, 0, 0, 0)
			w.renderer.Clear()
			w.update(w)

			w.Events = nil
			w.MouseMoved = false
			w.clicks = nil

			lastUpdateTime = now
			w.renderer.Present()
		} else {
			sdl.Delay(1)
		}
	}
}

func makeClick(event *sdl.MouseButtonEvent) MouseClick {
	return MouseClick{int(event.X), int(event.Y), MouseButton(event.Button)}
}

func (w *window) close() {
	for _, sound := range w.soundChunks {
		if sound != nil {
			sound.Free()
		}
	}
	for _, texture := range w.textures {
		if texture != nil {
			texture.Destroy()
		}
	}
	if w.fontTexture != nil {
		w.fontTexture.Destroy()
	}
}

func (w *window) Size() (int, int) {
	return w.window.GetSize()
}

func (w *window) WasKeyPressed(key string) bool {
	for _, e := range w.Events {
		switch event := e.(type) {
		case *sdl.KeyDownEvent:
			return isKey(key, event.Keysym.Sym)
		}
	}
	return false
}

func (w *window) setKeyDown(key sdl.Keycode, down bool) {
	name := strings.ToLower(keyToString[key])
	w.keyDown[name] = down
}

func (w *window) IsKeyDown(key string) bool {
	return w.keyDown[strings.ToLower(key)]
}

func isKey(name string, key sdl.Keycode) bool {
	keyString, ok := keyToString[key]
	return ok && strings.ToLower(keyString) == strings.ToLower(name)
}

var keyToString map[sdl.Keycode]string

func (w *window) IsMouseDown(button MouseButton) bool {
	return w.mouseDown[button]
}

func (w *window) Clicks() []MouseClick {
	return w.clicks
}

func (w *window) Close() {
	w.running = false
}

func (w *window) DrawEllipse(x, y, width, height int, color Color) {
	points := ellipsePoints(x, y, width, height)
	if len(points) > 0 {
		w.setColor(color)
		w.renderer.DrawPoints(makeSDLpoints(points))
	}
}

func (w *window) FillEllipse(x, y, width, height int, color Color) {
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

func (w *window) DrawPoint(x, y int, color Color) {
	w.setColor(color)
	w.renderer.DrawPoint(x, y)
}

func (w *window) setColor(color Color) {
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

func (w *window) DrawLine(fromX, fromY, toX, toY int, color Color) {
	w.setColor(color)
	w.line(fromX, fromY, toX, toY)
}

func (w *window) line(fromX, fromY, toX, toY int) {
	w.renderer.DrawLine(fromX, fromY, toX, toY)
}

func (w *window) DrawRect(x, y, width, height int, color Color) {
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

func (w *window) FillRect(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}
	w.setColor(color)
	w.renderer.FillRect(&sdl.Rect{int32(x), int32(y), int32(width), int32(height)})
}

func (w *window) DrawImageFile(path string, x, y int) error {
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

func (win *window) DrawImageFileTo(path string, x, y, w, h, degrees int) error {
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

func (w *window) DrawImageFilePortion(path string, srcX, srcY, srcW, srcH, toX, toY int) error {
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

func (w *window) loadImageIfNecessary(path string) {
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

func (win *window) GetTextSize(text string) (w, h int) {
	return win.GetScaledTextSize(text, 1.0)
}

func (win *window) GetScaledTextSize(text string, scale float32) (w, h int) {
	if len(text) == 0 {
		return 0, 0
	}
	_, _, width, height, _ := win.fontTexture.Query()
	width /= 16
	height /= 16
	w = int(float32(width)*scale + 0.5)
	h = int(float32(height)*scale + 0.5)
	lines := strings.Split(text, "\n")
	maxLineW := 0
	for _, line := range lines {
		if len(line) > maxLineW {
			maxLineW = len(line)
		}
	}
	return w * maxLineW, h * len(lines)
}

func (w *window) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1.0, color)
}

func (w *window) DrawScaledText(text string, x, y int, scale float32, color Color) {
	w.fontTexture.SetColorMod(channel(color.R), channel(color.G), channel(color.B))
	_, _, width, height, _ := w.fontTexture.Query()
	width /= 16
	height /= 16
	src := sdl.Rect{0, 0, int32(width), int32(height)}
	dest := src
	dest.W = int32(float32(dest.W)*scale + 0.5)
	dest.H = int32(float32(dest.H)*scale + 0.5)
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

func (w *window) PlaySoundFile(path string) error {
	w.loadSoundIfNecessary(path)
	sound := w.soundChunks[path]
	if sound == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	sound.PlayChannel(-1, 0)
	return nil
}

func (w *window) loadSoundIfNecessary(path string) {
	if _, ok := w.soundChunks[path]; ok {
		return
	}
	w.soundChunks[path], _ = mix.LoadWAV(path)
}

func init() {
	keyToString = make(map[sdl.Keycode]string)
	keyToString[sdl.K_UNKNOWN] = "UNKNOWN"
	keyToString[sdl.K_RETURN] = "RETURN"
	keyToString[sdl.K_ESCAPE] = "ESCAPE"
	keyToString[sdl.K_BACKSPACE] = "BACKSPACE"
	keyToString[sdl.K_TAB] = "TAB"
	keyToString[sdl.K_SPACE] = "SPACE"
	keyToString[sdl.K_COMMA] = "COMMA"
	keyToString[sdl.K_MINUS] = "MINUS"
	keyToString[sdl.K_PERIOD] = "PERIOD"
	keyToString[sdl.K_SLASH] = "SLASH"
	keyToString[sdl.K_0] = "0"
	keyToString[sdl.K_1] = "1"
	keyToString[sdl.K_2] = "2"
	keyToString[sdl.K_3] = "3"
	keyToString[sdl.K_4] = "4"
	keyToString[sdl.K_5] = "5"
	keyToString[sdl.K_6] = "6"
	keyToString[sdl.K_7] = "7"
	keyToString[sdl.K_8] = "8"
	keyToString[sdl.K_9] = "9"
	keyToString[sdl.K_SEMICOLON] = "SEMICOLON"
	keyToString[sdl.K_LEFTBRACKET] = "LEFTBRACKET"
	keyToString[sdl.K_BACKSLASH] = "BACKSLASH"
	keyToString[sdl.K_RIGHTBRACKET] = "RIGHTBRACKET"
	keyToString[sdl.K_a] = "a"
	keyToString[sdl.K_b] = "b"
	keyToString[sdl.K_c] = "c"
	keyToString[sdl.K_d] = "d"
	keyToString[sdl.K_e] = "e"
	keyToString[sdl.K_f] = "f"
	keyToString[sdl.K_g] = "g"
	keyToString[sdl.K_h] = "h"
	keyToString[sdl.K_i] = "i"
	keyToString[sdl.K_j] = "j"
	keyToString[sdl.K_k] = "k"
	keyToString[sdl.K_l] = "l"
	keyToString[sdl.K_m] = "m"
	keyToString[sdl.K_n] = "n"
	keyToString[sdl.K_o] = "o"
	keyToString[sdl.K_p] = "p"
	keyToString[sdl.K_q] = "q"
	keyToString[sdl.K_r] = "r"
	keyToString[sdl.K_s] = "s"
	keyToString[sdl.K_t] = "t"
	keyToString[sdl.K_u] = "u"
	keyToString[sdl.K_v] = "v"
	keyToString[sdl.K_w] = "w"
	keyToString[sdl.K_x] = "x"
	keyToString[sdl.K_y] = "y"
	keyToString[sdl.K_z] = "z"
	keyToString[sdl.K_CAPSLOCK] = "CAPSLOCK"
	keyToString[sdl.K_F1] = "F1"
	keyToString[sdl.K_F2] = "F2"
	keyToString[sdl.K_F3] = "F3"
	keyToString[sdl.K_F4] = "F4"
	keyToString[sdl.K_F5] = "F5"
	keyToString[sdl.K_F6] = "F6"
	keyToString[sdl.K_F7] = "F7"
	keyToString[sdl.K_F8] = "F8"
	keyToString[sdl.K_F9] = "F9"
	keyToString[sdl.K_F10] = "F10"
	keyToString[sdl.K_F11] = "F11"
	keyToString[sdl.K_F12] = "F12"
	keyToString[sdl.K_PRINTSCREEN] = "PRINTSCREEN"
	keyToString[sdl.K_SCROLLLOCK] = "SCROLLLOCK"
	keyToString[sdl.K_PAUSE] = "PAUSE"
	keyToString[sdl.K_INSERT] = "INSERT"
	keyToString[sdl.K_HOME] = "HOME"
	keyToString[sdl.K_PAGEUP] = "PAGEUP"
	keyToString[sdl.K_DELETE] = "DELETE"
	keyToString[sdl.K_END] = "END"
	keyToString[sdl.K_PAGEDOWN] = "PAGEDOWN"
	keyToString[sdl.K_RIGHT] = "RIGHT"
	keyToString[sdl.K_LEFT] = "LEFT"
	keyToString[sdl.K_DOWN] = "DOWN"
	keyToString[sdl.K_UP] = "UP"
	keyToString[sdl.K_KP_DIVIDE] = "KP_DIVIDE"
	keyToString[sdl.K_KP_MULTIPLY] = "KP_MULTIPLY"
	keyToString[sdl.K_KP_MINUS] = "KP_MINUS"
	keyToString[sdl.K_KP_PLUS] = "KP_PLUS"
	keyToString[sdl.K_KP_ENTER] = "KP_ENTER"
	keyToString[sdl.K_KP_1] = "KP_1"
	keyToString[sdl.K_KP_2] = "KP_2"
	keyToString[sdl.K_KP_3] = "KP_3"
	keyToString[sdl.K_KP_4] = "KP_4"
	keyToString[sdl.K_KP_5] = "KP_5"
	keyToString[sdl.K_KP_6] = "KP_6"
	keyToString[sdl.K_KP_7] = "KP_7"
	keyToString[sdl.K_KP_8] = "KP_8"
	keyToString[sdl.K_KP_9] = "KP_9"
	keyToString[sdl.K_KP_0] = "KP_0"
	keyToString[sdl.K_KP_PERIOD] = "KP_PERIOD"
	keyToString[sdl.K_KP_EQUALS] = "KP_EQUALS"
	keyToString[sdl.K_F13] = "F13"
	keyToString[sdl.K_F14] = "F14"
	keyToString[sdl.K_F15] = "F15"
	keyToString[sdl.K_F16] = "F16"
	keyToString[sdl.K_F17] = "F17"
	keyToString[sdl.K_F18] = "F18"
	keyToString[sdl.K_F19] = "F19"
	keyToString[sdl.K_F20] = "F20"
	keyToString[sdl.K_F21] = "F21"
	keyToString[sdl.K_F22] = "F22"
	keyToString[sdl.K_F23] = "F23"
	keyToString[sdl.K_F24] = "F24"
	keyToString[sdl.K_MENU] = "MENU"
	keyToString[sdl.K_KP_COMMA] = "KP_COMMA"
	keyToString[sdl.K_LCTRL] = "LCTRL"
	keyToString[sdl.K_LSHIFT] = "LSHIFT"
	keyToString[sdl.K_LALT] = "LALT"
	keyToString[sdl.K_LGUI] = "LGUI"
	keyToString[sdl.K_RCTRL] = "RCTRL"
	keyToString[sdl.K_RSHIFT] = "RSHIFT"
	keyToString[sdl.K_RALT] = "RALT"
	keyToString[sdl.K_RGUI] = "RGUI"
}
