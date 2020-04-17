// +build sdl2,!glfw

package draw

import (
	"errors"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/gonutz/go-sdl2/img"
	"github.com/gonutz/go-sdl2/mix"
	"github.com/gonutz/go-sdl2/sdl"
)

func init() {
	runtime.LockOSThread()
}

type window struct {
	update      UpdateFunction
	window      *sdl.Window
	running     bool
	renderer    *sdl.Renderer
	textures    map[string]*sdl.Texture
	soundChunks map[string]*mix.Chunk
	fontTexture *sdl.Texture
	keyDown     map[Key]bool
	typed       []rune
	mouseDown   map[MouseButton]bool
	clicks      []MouseClick
	pressedKeys []Key
	mouse       struct{ x, y int }
	wheelX      float64
	wheelY      float64
}

var windowRunningMutex sync.Mutex

// RunWindow creates a new window and calls update 60 times per second.
func RunWindow(title string, width, height int, update UpdateFunction) error {
	windowRunningMutex.Lock()

	if update == nil {
		return errors.New("Update function was nil.")
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return err
	}
	defer sdl.Quit()

	sdlWindow, renderer, err := sdl.CreateWindowAndRenderer(int32(width), int32(height), 0)
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
		keyDown:     make(map[Key]bool),
		mouseDown:   make(map[MouseButton]bool),
	}
	win.createBitmapFont()
	win.runMainLoop()
	win.close()

	windowRunningMutex.Unlock()
	return nil
}

func (w *window) createBitmapFont() {
	rwops, _ := sdl.RWFromMem(bitmapFontWhitePng)
	texture, err := img.LoadTextureRW(w.renderer, rwops, false)
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
			case *sdl.TextInputEvent:
				textLength := 0
				for i, b := range event.Text {
					if b == 0 {
						textLength = i
						break
					}
				}
				text := string(event.Text[:textLength])
				for _, r := range text {
					w.typed = append(w.typed, r)
				}
			case *sdl.MouseMotionEvent:
				w.mouse.x = int(event.X)
				w.mouse.y = int(event.Y)
			case *sdl.MouseButtonEvent:
				if event.State == sdl.PRESSED {
					w.clicks = append(w.clicks, makeClick(event))
					w.mouseDown[MouseButton(event.Button)] = true
				}
				if event.State == sdl.RELEASED {
					w.mouseDown[MouseButton(event.Button)] = false
				}
			case *sdl.MouseWheelEvent:
				dx, dy := event.X, event.Y
				if event.Direction == sdl.MOUSEWHEEL_FLIPPED {
					dx, dy = -event.X, -event.Y
				}
				w.wheelX += float64(dx)
				w.wheelY += float64(dy)
			case *sdl.KeyboardEvent:
				if event.Type == sdl.KEYDOWN {
					w.setKeyDown(event.Keysym.Sym, true)
				} else {
					w.setKeyDown(event.Keysym.Sym, false)
				}
			}
		}

		now := time.Now()
		if now.Sub(lastUpdateTime).Seconds() > updateInterval {
			// clear background to black
			w.renderer.SetDrawColor(0, 0, 0, 0)
			w.renderer.Clear()
			// client updates window
			w.update(w)
			// reset all events
			w.pressedKeys = nil
			w.clicks = nil
			w.typed = nil
			w.wheelX = 0
			w.wheelY = 0
			lastUpdateTime = now
			// show the window
			w.renderer.Present()
		} else {
			sdl.Delay(1)
		}
	}
}

func makeClick(event *sdl.MouseButtonEvent) MouseClick {
	return MouseClick{int(event.X), int(event.Y), MouseButton(event.Button)}
}

func (w *window) setKeyDown(key sdl.Keycode, down bool) {
	k := toKey(key)
	if k != 0 {
		w.keyDown[k] = down
		if down {
			w.pressedKeys = append(w.pressedKeys, k)
		}
	}
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
	width, height := w.window.GetSize()
	return int(width), int(height)
}

func (w *window) SetFullscreen(f bool) {
	if f {
		w.window.SetFullscreen(sdl.WINDOW_FULLSCREEN_DESKTOP)
	} else {
		w.window.SetFullscreen(0)
	}
}

func (w *window) WasKeyPressed(key Key) bool {
	for _, k := range w.pressedKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (w *window) IsKeyDown(key Key) bool {
	return w.keyDown[key]
}

func (w *window) WasCharTyped(char rune) bool {
	for _, r := range w.typed {
		if char == r {
			return true
		}
	}
	return false
}

func (w *window) IsMouseDown(button MouseButton) bool {
	return w.mouseDown[button]
}

func (w *window) Clicks() []MouseClick {
	return w.clicks
}

func (w *window) Characters() string {
	return string(w.typed)
}

func (w *window) MousePosition() (int, int) {
	return w.mouse.x, w.mouse.y
}

func (w *window) MouseWheelY() float64 {
	return w.wheelY
}

func (w *window) MouseWheelX() float64 {
	return w.wheelX
}

func (w *window) Close() {
	w.running = false
}

func (w *window) DrawEllipse(x, y, width, height int, color Color) {
	outline := ellipseOutline(x, y, width, height)
	if len(outline) > 0 {
		w.setColor(color)
		w.renderer.DrawPoints(makeSDLpoints(outline))
	}
}

func (w *window) FillEllipse(x, y, width, height int, color Color) {
	if width == 1 && height == 1 {
		w.DrawPoint(x, y, color)
		return
	}
	points := ellipseArea(x, y, width, height)
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

func (w *window) DrawPoint(x, y int, color Color) {
	w.setColor(color)
	w.renderer.DrawPoint(int32(x), int32(y))
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
	w.renderer.DrawLine(int32(fromX), int32(fromY), int32(toX), int32(toY))
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

func (win *window) DrawImageFileRotated(path string, x, y, degrees int) error {
	win.loadImageIfNecessary(path)
	img := win.textures[path]
	if img == nil {
		return errors.New(`File "` + path + `" could not be loaded.`)
	}
	_, _, width, height, _ := img.Query()
	win.renderer.CopyEx(
		img,
		nil,
		&sdl.Rect{int32(x), int32(y), width, height},
		float64(degrees),
		nil,
		sdl.FLIP_NONE)
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
	_, _, width, height, _ := win.fontTexture.Query()
	width /= 16
	height /= 16
	w = int(float32(width)*scale + 0.5)
	h = int(float32(height)*scale + 0.5)
	lines := strings.Split(text, "\n")
	maxLineW := 0
	for _, line := range lines {
		w := utf8.RuneCountInString(line)
		if w > maxLineW {
			maxLineW = w
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
	for _, r := range text {
		if r == '\n' {
			dest.X = int32(x)
			dest.Y += dest.H
			continue
		}
		r = runeToFont(r)
		src.X = int32(r%16) * width
		src.Y = int32(r/16) * height
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
	sound.Play(-1, 0)
	return nil
}

func (w *window) loadSoundIfNecessary(path string) {
	if _, ok := w.soundChunks[path]; ok {
		return
	}
	w.soundChunks[path], _ = mix.LoadWAV(path)
}

func toKey(k sdl.Keycode) Key {
	switch k {
	case sdl.K_a:
		return KeyA
	case sdl.K_b:
		return KeyB
	case sdl.K_c:
		return KeyC
	case sdl.K_d:
		return KeyD
	case sdl.K_e:
		return KeyE
	case sdl.K_f:
		return KeyF
	case sdl.K_g:
		return KeyG
	case sdl.K_h:
		return KeyH
	case sdl.K_i:
		return KeyI
	case sdl.K_j:
		return KeyJ
	case sdl.K_k:
		return KeyK
	case sdl.K_l:
		return KeyL
	case sdl.K_m:
		return KeyM
	case sdl.K_n:
		return KeyN
	case sdl.K_o:
		return KeyO
	case sdl.K_p:
		return KeyP
	case sdl.K_q:
		return KeyQ
	case sdl.K_r:
		return KeyR
	case sdl.K_s:
		return KeyS
	case sdl.K_t:
		return KeyT
	case sdl.K_u:
		return KeyU
	case sdl.K_v:
		return KeyV
	case sdl.K_w:
		return KeyW
	case sdl.K_x:
		return KeyX
	case sdl.K_y:
		return KeyY
	case sdl.K_z:
		return KeyZ
	case sdl.K_0:
		return Key0
	case sdl.K_1:
		return Key1
	case sdl.K_2:
		return Key2
	case sdl.K_3:
		return Key3
	case sdl.K_4:
		return Key4
	case sdl.K_5:
		return Key5
	case sdl.K_6:
		return Key6
	case sdl.K_7:
		return Key7
	case sdl.K_8:
		return Key8
	case sdl.K_9:
	case sdl.K_KP_0:
		return KeyNum0
	case sdl.K_KP_1:
		return KeyNum1
	case sdl.K_KP_2:
		return KeyNum2
	case sdl.K_KP_3:
		return KeyNum3
	case sdl.K_KP_4:
		return KeyNum4
	case sdl.K_KP_5:
		return KeyNum5
	case sdl.K_KP_6:
		return KeyNum6
	case sdl.K_KP_7:
		return KeyNum7
	case sdl.K_KP_8:
		return KeyNum8
	case sdl.K_KP_9:
		return KeyNum9
	case sdl.K_F1:
		return KeyF1
	case sdl.K_F2:
		return KeyF2
	case sdl.K_F3:
		return KeyF3
	case sdl.K_F4:
		return KeyF4
	case sdl.K_F5:
		return KeyF5
	case sdl.K_F6:
		return KeyF6
	case sdl.K_F7:
		return KeyF7
	case sdl.K_F8:
		return KeyF8
	case sdl.K_F9:
		return KeyF9
	case sdl.K_F10:
		return KeyF10
	case sdl.K_F11:
		return KeyF11
	case sdl.K_F12:
		return KeyF12
	case sdl.K_F13:
		return KeyF13
	case sdl.K_F14:
		return KeyF14
	case sdl.K_F15:
		return KeyF15
	case sdl.K_F16:
		return KeyF16
	case sdl.K_F17:
		return KeyF17
	case sdl.K_F18:
		return KeyF18
	case sdl.K_F19:
		return KeyF19
	case sdl.K_F20:
		return KeyF20
	case sdl.K_F21:
		return KeyF21
	case sdl.K_F22:
		return KeyF22
	case sdl.K_F23:
		return KeyF23
	case sdl.K_F24:
		return KeyF24
	case sdl.K_RETURN:
		return KeyEnter
	case sdl.K_KP_ENTER:
		return KeyNumEnter
	case sdl.K_LCTRL:
		return KeyLeftControl
	case sdl.K_RCTRL:
		return KeyRightControl
	case sdl.K_LSHIFT:
		return KeyLeftShift
	case sdl.K_RSHIFT:
		return KeyRightShift
	case sdl.K_LALT:
		return KeyLeftAlt
	case sdl.K_RALT:
		return KeyRightAlt
	case sdl.K_LEFT:
		return KeyLeft
	case sdl.K_RIGHT:
		return KeyRight
	case sdl.K_UP:
		return KeyUp
	case sdl.K_DOWN:
		return KeyDown
	case sdl.K_ESCAPE:
		return KeyEscape
	case sdl.K_SPACE:
		return KeySpace
	case sdl.K_BACKSPACE:
		return KeyBackspace
	case sdl.K_TAB:
		return KeyTab
	case sdl.K_HOME:
		return KeyHome
	case sdl.K_END:
		return KeyEnd
	case sdl.K_PAGEDOWN:
		return KeyPageDown
	case sdl.K_PAGEUP:
		return KeyPageUp
	case sdl.K_DELETE:
		return KeyDelete
	case sdl.K_INSERT:
		return KeyInsert
	case sdl.K_KP_PLUS:
		return KeyNumAdd
	case sdl.K_KP_MINUS:
		return KeyNumSubtract
	case sdl.K_KP_MULTIPLY:
		return KeyNumMultiply
	case sdl.K_KP_DIVIDE:
		return KeyNumDivide
	case sdl.K_CAPSLOCK:
		return KeyCapslock
	case sdl.K_PRINTSCREEN:
		return KeyPrint
	case sdl.K_PAUSE:
		return KeyPause
	}

	return Key(0)
}
