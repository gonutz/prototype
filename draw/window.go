package draw

import (
	"errors"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
	"strings"
	"sync"
	"time"
	"unsafe"
)

// UpdateFunction is used as a callback when creating a window. It is called
// regularly and you can do all your event handling and drawing in it.
type UpdateFunction func(window *Window)

type Window struct {
	Running    bool
	Events     []sdl.Event
	MouseMoved bool
	Mouse      struct{ X, Y int }
	Clicks     []MouseClick

	update      UpdateFunction
	window      *sdl.Window
	renderer    *sdl.Renderer
	textures    map[string]*sdl.Texture
	fontTexture *sdl.Texture
	keyDown     map[string]bool
	mouseDown   map[MouseButton]bool
}

type MouseClick struct {
	X, Y   int
	Button MouseButton
}

type MouseButton uint8

const (
	LeftButton   MouseButton = sdl.BUTTON_LEFT
	MiddleButton             = sdl.BUTTON_MIDDLE
	RightButton              = sdl.BUTTON_RIGHT
)

var windowRunningMutex sync.Mutex

func RunWindow(title string, width, height int, update UpdateFunction) error {
	windowRunningMutex.Lock()

	if update == nil {
		return errors.New("Update function was nil.")
	}

	if err := sdl.Init(sdl.INIT_EVERYTHING); err != nil {
		return err
	}
	defer sdl.Quit()

	window, renderer, err := sdl.CreateWindowAndRenderer(width, height, 0)
	if err != nil {
		return err
	}
	defer window.Destroy()
	defer renderer.Destroy()
	window.SetTitle(title)
	renderer.SetDrawBlendMode(sdl.BLENDMODE_BLEND)

	win := &Window{
		Running:   true,
		update:    update,
		window:    window,
		renderer:  renderer,
		textures:  make(map[string]*sdl.Texture),
		keyDown:   make(map[string]bool),
		mouseDown: make(map[MouseButton]bool),
	}
	win.createBitmapFont()
	win.runMainLoop()

	windowRunningMutex.Unlock()
	return nil
}

func (w *Window) createBitmapFont() {
	ptr := unsafe.Pointer(&bitmapFontWhitePng[0])
	rwops := sdl.RWFromMem(ptr, len(bitmapFontWhitePng))
	texture, err := img.LoadTexture_RW(w.renderer, rwops, 0)
	if err != nil {
		panic(err)
	}
	w.fontTexture = texture
}

func (w *Window) runMainLoop() {
	w.renderer.SetDrawColor(0, 0, 0, 0)

	lastUpdateTime := time.Now().Add(-time.Hour)
	const updateInterval = 1.0 / 60.0
	for w.Running {
		for e := sdl.PollEvent(); e != nil; e = sdl.PollEvent() {
			switch event := e.(type) {
			case *sdl.QuitEvent:
				w.Running = false
			case *sdl.MouseMotionEvent:
				w.Mouse.X = int(event.X)
				w.Mouse.Y = int(event.Y)
				w.MouseMoved = true
			case *sdl.MouseButtonEvent:
				if event.State == sdl.PRESSED {
					w.Clicks = append(w.Clicks, makeClick(event))
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
			w.Clicks = nil

			lastUpdateTime = now
			w.renderer.Present()
		}
	}

	w.close()
}

func makeClick(event *sdl.MouseButtonEvent) MouseClick {
	return MouseClick{int(event.X), int(event.Y), MouseButton(event.Button)}
}

func (w *Window) close() {
	for _, texture := range w.textures {
		if texture != nil {
			texture.Destroy()
		}
	}
	if w.fontTexture != nil {
		w.fontTexture.Destroy()
	}
}

func (w *Window) WasKeyPressed(key string) bool {
	for _, e := range w.Events {
		switch event := e.(type) {
		case *sdl.KeyDownEvent:
			return isKey(key, event.Keysym.Sym)
		}
	}
	return false
}

func (w *Window) setKeyDown(key sdl.Keycode, down bool) {
	name := strings.ToLower(keyToString[key])
	w.keyDown[name] = down
}

func (w *Window) IsKeyDown(key string) bool {
	return w.keyDown[strings.ToLower(key)]
}

func isKey(name string, key sdl.Keycode) bool {
	keyString, ok := keyToString[key]
	return ok && strings.ToLower(keyString) == strings.ToLower(name)
}

var keyToString map[sdl.Keycode]string

func (w *Window) IsMouseDown(button MouseButton) bool {
	return w.mouseDown[button]
}
