//go:build js && wasm
// +build js,wasm

package draw

import (
	"fmt"
	"io"
	"math"
	"strings"
	"syscall/js"
	"time"

	_ "embed"
)

//go:embed Go-Mono.ttf
var fontData []byte

type wasmWindow struct {
	canvas           js.Value
	ctx              js.Value
	width            int
	height           int
	running          bool
	showingCursor    bool
	keyDown          [keyCount]bool
	pressedKeys      []Key
	typed            string
	mouseX           int
	mouseY           int
	mouseDown        [mouseButtonCount]bool
	wheelX           float64
	wheelY           float64
	clicks           []MouseClick
	images           map[string]*imageState
	audioCtx         js.Value
	audioBuffers     map[string]js.Value
	fontURL          js.Value
	wantFullscreen   bool
	isFullscreen     bool
	hasSeenUserInput bool
	soundsToPlay     []futureSound
	iconPath         string
}

type imageState struct {
	image js.Value
	err   error
}

type futureSound struct {
	source    js.Value
	startedAt time.Time
}

func RunWindow(title string, width, height int, update UpdateFunction) error {
	doc := js.Global().Get("document")
	doc.Set("title", title)
	canvas := doc.Call("getElementById", "gameCanvas")
	if !canvas.Truthy() {
		return js.Error{Value: js.ValueOf("canvas element not found")}
	}
	canvas.Set("width", width)
	canvas.Set("height", height)

	window := &wasmWindow{
		running:       true,
		width:         width,
		height:        height,
		showingCursor: true,
		canvas:        canvas,
		ctx:           canvas.Call("getContext", "2d"),
		audioCtx:      js.Global().Get("AudioContext").New(),
		images:        map[string]*imageState{},
		audioBuffers:  map[string]js.Value{},
	}

	defer window.ShowCursor(true)

	bindEvent(js.Global(), "keydown", func(e js.Value) {
		if !window.running {
			return
		}

		window.onUserInteraction()

		keyCode := e.Get("code").String()
		keyValue := e.Get("key").String()
		key := toKey(keyCode, keyValue)

		if key != 0 && !window.keyDown[key] {
			window.pressedKeys = append(window.pressedKeys, key)
		}
		window.keyDown[key] = true

		if window.keyDown[KeyLeftControl] || window.keyDown[KeyRightControl] ||
			window.keyDown[KeyLeftAlt] || window.keyDown[KeyRightAlt] ||
			preventKeyDownDefault[key] {
			e.Call("preventDefault")
		}
	})

	bindEvent(js.Global(), "keyup", func(e js.Value) {
		if !window.running {
			return
		}

		keyCode := e.Get("code").String()
		keyValue := e.Get("key").String()
		key := toKey(keyCode, keyValue)
		if key != 0 {
			window.keyDown[key] = false
		}
	})

	bindEvent(js.Global(), "keypress", func(e js.Value) {
		if !window.running {
			return
		}

		window.onUserInteraction()

		key := e.Get("key").String()
		if key != "Enter" && len(key) > 0 {
			window.typed += key
		}
	})

	bindEvent(doc, "mousemove", func(e js.Value) {
		if !window.running {
			return
		}

		bounds := canvas.Call("getBoundingClientRect")
		window.mouseX = e.Get("clientX").Int() - bounds.Get("left").Int()
		window.mouseY = e.Get("clientY").Int() - bounds.Get("top").Int()
	})

	// To determine whether the mouse buttons are currently up or down, we
	// register the mouse down and up events on the *document*.
	// To collect mouse clicks, we register the mouse down event on the
	// *canvas*. Clicks outside the canvas are not reported.
	bindEvent(doc, "mousedown", func(e js.Value) {
		if !window.running {
			return
		}

		window.onUserInteraction()

		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			window.mouseDown[button] = true
		}

		e.Call("preventDefault")
	})
	bindEvent(doc, "mouseup", func(e js.Value) {
		if !window.running {
			return
		}

		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			window.mouseDown[button] = false
		}
	})
	bindEvent(canvas, "mousedown", func(e js.Value) {
		if !window.running {
			return
		}

		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			window.clicks = append(window.clicks, MouseClick{
				X:      window.mouseX,
				Y:      window.mouseY,
				Button: MouseButton(button),
			})
		}
	})

	bindEvent(canvas, "wheel", func(e js.Value) {
		if !window.running {
			return
		}

		window.wheelX -= e.Get("deltaX").Float() / 100
		window.wheelY -= e.Get("deltaY").Float() / 100
		e.Call("preventDefault")
	})

	// Suppress right clicks triggering the context menu.
	bindEvent(canvas, "contextmenu", func(e js.Value) {
		if !window.running {
			return
		}

		e.Call("preventDefault")
	})

	bindEvent(doc, "fullscreenchange", func(e js.Value) {
		if !window.running {
			return
		}

		window.isFullscreen = doc.Get("fullscreenElement").Truthy()
		window.wantFullscreen = window.isFullscreen
		if window.isFullscreen {
			win := js.Global().Get("window")
			canvas.Set("width", win.Get("innerWidth"))
			canvas.Set("height", win.Get("innerHeight"))
			canvas.Get("style").Set("width", "100vw")
			canvas.Get("style").Set("height", "100vh")
		} else {
			canvas.Set("width", width)
			canvas.Set("height", height)
			canvas.Get("style").Set("width", fmt.Sprintf("%dpx", width))
			canvas.Get("style").Set("height", fmt.Sprintf("%dpx", height))
		}
	})

	fontArray := js.Global().Get("Uint8Array").New(len(fontData))
	js.CopyBytesToJS(fontArray, fontData)
	fontBlob := js.Global().Get("Blob").New(js.ValueOf([]interface{}{fontArray}))
	window.fontURL = js.Global().Get("URL").Call("createObjectURL", fontBlob)
	style := js.Global().Get("document").Call("createElement", "style")
	style.Set("textContent", "@font-face { font-family: '_draw_font_'; src: url('"+window.fontURL.String()+"'); }")
	js.Global().Get("document").Get("head").Call("appendChild", style)
	js.Global().Get("document").Get("fonts").Call("load", "1em _draw_font_")

	// Main render loop using requestAnimationFrame.
	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		window.FillRect(0, 0, 99999, 99999, Black)
		if window.running {
			update(window)
			// Reset input state between frames.
			window.wheelX = 0
			window.wheelY = 0
			window.clicks = window.clicks[:0]
			window.pressedKeys = window.pressedKeys[:0]
			window.typed = ""
			js.Global().Call("requestAnimationFrame", renderFrame)
		}
		return nil
	})
	js.Global().Call("requestAnimationFrame", renderFrame)

	// WASM requires us to prevent main from exiting.
	select {}
}

func bindEvent(target js.Value, event string, handler func(js.Value)) js.Func {
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(args[0])
		return nil
	})
	target.Call("addEventListener", event, jsFunc)
	return jsFunc
}

func (w *wasmWindow) startAudioPlayback() {
	if w.audioCtx.Get("state").String() == "suspended" {
		promise := w.audioCtx.Call("resume")

		// Play all the sounds that have been started before sound was
		// available. Play them at their offset relative to when they were
		// started.
		promise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			now := time.Now()
			for _, s := range w.soundsToPlay {
				offset := now.Sub(s.startedAt).Seconds()
				s.source.Call("start", 0, offset)
			}

			w.soundsToPlay = nil
			return nil
		}))
	}
}

func (w *wasmWindow) setColor(c Color) {
	r := int(c.R * 255)
	g := int(c.G * 255)
	b := int(c.B * 255)
	a := c.A
	// We use CSS-style RGBA strings.
	col := fmt.Sprintf("rgba(%d,%d,%d,%f)", r, g, b, a)
	w.ctx.Set("fillStyle", col)
	w.ctx.Set("strokeStyle", col)
}

func (w *wasmWindow) loadImage(path string) (js.Value, error) {
	// There are 4 possible image states:
	// 1. Never seen before - must be loaded.
	// 2. Loading has started and not yet finished.
	// 3. Loading was successful - return the cached image.
	// 4. Loading failed - return the cached error.

	if imgState, ok := w.images[path]; ok {
		return imgState.image, imgState.err
	}

	img := js.Global().Get("Image").New()

	w.images[path] = &imageState{
		image: img,
		err:   ErrImageLoading,
	}

	img.Set("onload", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.images[path].err = nil
		return nil
	}))

	img.Set("onerror", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.images[path].err = fmt.Errorf("failed to load image \"%s\"", path)
		return nil
	}))

	if OpenFile != nil {
		url, err := loadBlob(path)
		if err != nil {
			w.images[path].err = err
		} else {
			img.Set("src", url)
		}
	} else {
		img.Set("src", path)
	}

	imgState := w.images[path]
	return imgState.image, imgState.err
}

func loadBlob(path string) (js.Value, error) {
	f, err := OpenFile(path)
	if err != nil {
		return js.Null(), err
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return js.Null(), err
	}

	array := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(array, data)

	blob := js.Global().Get("Blob").New([]interface{}{array})
	url := js.Global().Get("URL").Call("createObjectURL", blob)

	return url, nil
}

var keyMap = map[string]Key{
	"KeyA":           KeyA,
	"KeyB":           KeyB,
	"KeyC":           KeyC,
	"KeyD":           KeyD,
	"KeyE":           KeyE,
	"KeyF":           KeyF,
	"KeyG":           KeyG,
	"KeyH":           KeyH,
	"KeyI":           KeyI,
	"KeyJ":           KeyJ,
	"KeyK":           KeyK,
	"KeyL":           KeyL,
	"KeyM":           KeyM,
	"KeyN":           KeyN,
	"KeyO":           KeyO,
	"KeyP":           KeyP,
	"KeyQ":           KeyQ,
	"KeyR":           KeyR,
	"KeyS":           KeyS,
	"KeyT":           KeyT,
	"KeyU":           KeyU,
	"KeyV":           KeyV,
	"KeyW":           KeyW,
	"KeyX":           KeyX,
	"KeyY":           KeyY,
	"KeyZ":           KeyZ,
	"Digit0":         Key0,
	"Digit1":         Key1,
	"Digit2":         Key2,
	"Digit3":         Key3,
	"Digit4":         Key4,
	"Digit5":         Key5,
	"Digit6":         Key6,
	"Digit7":         Key7,
	"Digit8":         Key8,
	"Digit9":         Key9,
	"Numpad0":        KeyNum0,
	"Numpad1":        KeyNum1,
	"Numpad2":        KeyNum2,
	"Numpad3":        KeyNum3,
	"Numpad4":        KeyNum4,
	"Numpad5":        KeyNum5,
	"Numpad6":        KeyNum6,
	"Numpad7":        KeyNum7,
	"Numpad8":        KeyNum8,
	"Numpad9":        KeyNum9,
	"F1":             KeyF1,
	"F2":             KeyF2,
	"F3":             KeyF3,
	"F4":             KeyF4,
	"F5":             KeyF5,
	"F6":             KeyF6,
	"F7":             KeyF7,
	"F8":             KeyF8,
	"F9":             KeyF9,
	"F10":            KeyF10,
	"F11":            KeyF11,
	"F12":            KeyF12,
	"F13":            KeyF13,
	"F14":            KeyF14,
	"F15":            KeyF15,
	"F16":            KeyF16,
	"F17":            KeyF17,
	"F18":            KeyF18,
	"F19":            KeyF19,
	"F20":            KeyF20,
	"F21":            KeyF21,
	"F22":            KeyF22,
	"F23":            KeyF23,
	"F24":            KeyF24,
	"Enter":          KeyEnter,
	"NumpadEnter":    KeyNumEnter,
	"ControlLeft":    KeyLeftControl,
	"ControlRight":   KeyRightControl,
	"ShiftLeft":      KeyLeftShift,
	"ShiftRight":     KeyRightShift,
	"AltLeft":        KeyLeftAlt,
	"AltRight":       KeyRightAlt,
	"ArrowLeft":      KeyLeft,
	"ArrowRight":     KeyRight,
	"ArrowUp":        KeyUp,
	"ArrowDown":      KeyDown,
	"Escape":         KeyEscape,
	"Space":          KeySpace,
	"Backspace":      KeyBackspace,
	"Tab":            KeyTab,
	"Home":           KeyHome,
	"End":            KeyEnd,
	"PageDown":       KeyPageDown,
	"PageUp":         KeyPageUp,
	"Delete":         KeyDelete,
	"Insert":         KeyInsert,
	"NumpadAdd":      KeyNumAdd,
	"NumpadSubtract": KeyNumSubtract,
	"NumpadMultiply": KeyNumMultiply,
	"NumpadDivide":   KeyNumDivide,
	"CapsLock":       KeyCapslock,
	"Pause":          KeyPause,
	"PrintScreen":    KeyPrint,
}

var preventKeyDownDefault = map[Key]bool{
	KeyF1:           true,
	KeyF2:           true,
	KeyF3:           true,
	KeyF4:           true,
	KeyF5:           true,
	KeyF6:           true,
	KeyF7:           true,
	KeyF8:           true,
	KeyF9:           true,
	KeyF10:          true,
	KeyF11:          true,
	KeyF12:          true,
	KeyF13:          true,
	KeyF14:          true,
	KeyF15:          true,
	KeyF16:          true,
	KeyF17:          true,
	KeyF18:          true,
	KeyF19:          true,
	KeyF20:          true,
	KeyF21:          true,
	KeyF22:          true,
	KeyF23:          true,
	KeyF24:          true,
	KeyLeftControl:  true,
	KeyRightControl: true,
	KeyLeftShift:    true,
	KeyRightShift:   true,
	KeyLeftAlt:      true,
	KeyRightAlt:     true,
	KeyTab:          true,
	KeyHome:         true,
	KeyEnd:          true,
	KeyPageDown:     true,
	KeyPageUp:       true,
	KeyCapslock:     true,
	KeyPrint:        true,
	KeyPause:        true,
}

func toKey(code, value string) Key {
	// JavaScript's keydown event gives us a key code and a key value. The key
	// code is key layout independent. The key value represents the character on
	// the key. Take for example a German keyboard where - compared to a US
	// keyboard - the Z and Y keys are swapped. Here the key code for the Key
	// between T and U, which on the German keyboard is the Z, will be "KeyY"
	// while the key value will be "z" or "Z", depending on whether shift is
	// held at the time of the key press.
	// To replicate the behavior on the desktop, we need to handle the German Z
	// key as KeyZ, even though JS gives us code KeyY for it. We use a
	// combination of key code and key value to differentiate these.
	if strings.HasPrefix(code, "Key") {
		k := strings.TrimPrefix(code, "Key")
		if isUpperCaseLetter(k) {
			// Key code is in [KeyA..KeyZ].
			v := strings.ToUpper(value)
			if isUpperCaseLetter(v) {
				// Key value converted to upper-case is in [A..Z].
				return KeyA + Key(v[0]-'A')
			}
		}
	}

	return keyMap[code] // Defaults to 0 which is good.
}

func isUpperCaseLetter(s string) bool {
	return len(s) == 1 && 'A' <= s[0] && s[0] <= 'Z'
}

func (w *wasmWindow) Close() {
	w.running = false
	w.canvas.Get("style").Set("cursor", "default")
	if w.isFullscreen {
		js.Global().Get("document").Call("exitFullscreen")
	}
	w.audioCtx.Call("close")
}

func (w *wasmWindow) SetIcon(path string) error {
	if w.iconPath == path {
		return nil
	}

	doc := js.Global().Get("document")
	link := doc.Call("querySelector", "link[rel~='icon']")
	if !link.Truthy() {
		link = doc.Call("createElement", "link")
		link.Set("rel", "icon")
		link = doc.Get("head").Call("appendChild", link)
	}

	if OpenFile != nil {
		url, err := loadBlob(path)
		if err != nil {
			return err
		}
		link.Set("href", url)
	} else {
		link.Set("href", path)
	}

	w.iconPath = path

	return nil
}

func (w *wasmWindow) Size() (int, int) {
	return w.canvas.Get("width").Int(), w.canvas.Get("height").Int()
}

func (w *wasmWindow) onUserInteraction() {
	// In the browser, we need a user action to be allowed to go fullscreen
	// and play sounds so we do this in the key and mouse button handlers.
	if !w.hasSeenUserInput {
		w.hasSeenUserInput = true
		w.updateFullscreen()
		w.startAudioPlayback()
	}
}

func (w *wasmWindow) IsFullscreen() bool {
	return w.isFullscreen
}

func (w *wasmWindow) SetFullscreen(fullscreen bool) {
	w.wantFullscreen = fullscreen
	if w.hasSeenUserInput {
		w.updateFullscreen()
	}
}

func (w *wasmWindow) updateFullscreen() {
	if w.isFullscreen != w.wantFullscreen {
		if w.wantFullscreen {
			w.canvas.Call("requestFullscreen")
		} else {
			js.Global().Get("document").Call("exitFullscreen")
		}
	}
}

func (w *wasmWindow) ShowCursor(show bool) {
	if show == w.showingCursor {
		return
	}

	if show {
		w.canvas.Get("style").Set("cursor", "default")
	} else {
		w.canvas.Get("style").Set("cursor", "none")
	}

	w.showingCursor = show
}

func (w *wasmWindow) WasKeyPressed(key Key) bool {
	for _, k := range w.pressedKeys {
		if k == key {
			return true
		}
	}
	return false
}

func (w *wasmWindow) IsKeyDown(key Key) bool {
	return w.keyDown[key]
}

func (w *wasmWindow) Characters() string {
	return w.typed
}

func (w *wasmWindow) IsMouseDown(button MouseButton) bool {
	return w.mouseDown[button]
}

func (w *wasmWindow) Clicks() []MouseClick {
	return w.clicks
}

func (w *wasmWindow) MousePosition() (int, int) {
	return w.mouseX, w.mouseY
}

func (w *wasmWindow) MouseWheelX() float64 {
	return w.wheelX
}

func (w *wasmWindow) MouseWheelY() float64 {
	return w.wheelY
}

func (w *wasmWindow) DrawPoint(x, y int, c Color) {
	w.setColor(c)
	w.ctx.Call("fillRect", x, y, 1, 1)
}

func (w *wasmWindow) DrawLine(x1, y1, x2, y2 int, c Color) {
	w.setColor(c)

	// For extra nice pixels without the anti-aliasing, we use the Bresenham
	// line drawing algorithm. This makes the web lines look the same as the
	// desktop lines: pixelated.

	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

	sx := -1
	if x1 < x2 {
		sx = 1
	}

	sy := -1
	if y1 < y2 {
		sy = 1
	}

	err := dx - dy

	for {
		if x1 == x2 && y1 == y2 {
			break
		}
		w.ctx.Call("fillRect", x1, y1, 1, 1)
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (w *wasmWindow) DrawRect(x, y, width, height int, c Color) {
	if height == 1 {
		w.DrawLine(x, y, x+width, y, c)
	} else if width == 1 {
		w.DrawLine(x, y, x, y+height, c)
	} else if width > 0 && height > 0 {
		w.setColor(c)
		w.ctx.Call("strokeRect", float32(x)+0.5, float32(y)+0.5, width-1, height-1)
	}
}

func (w *wasmWindow) FillRect(x, y, width, height int, c Color) {
	if width <= 0 || height <= 0 {
		return
	}

	w.setColor(c)
	w.ctx.Call("fillRect", x, y, width, height)
}

func (w *wasmWindow) DrawEllipse(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}

	outline := ellipseOutline(x, y, width, height)
	if len(outline) == 0 {
		return
	}

	w.setColor(color)
	for _, p := range outline {
		w.ctx.Call("fillRect", p.x, p.y, 1, 1)
	}
}

func (w *wasmWindow) FillEllipse(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}

	area := ellipseArea(x, y, width, height)
	if len(area) == 0 {
		return
	}

	w.setColor(color)
	for len(area) > 1 {
		start, end := area[0], area[1]
		area = area[2:]
		w.ctx.Call("fillRect", start.x, start.y, end.x-start.x+1, 1)
	}
}

func (w *wasmWindow) ImageSize(path string) (int, int, error) {
	img, err := w.loadImage(path)
	if err != nil {
		return 0, 0, err
	}
	return img.Get("width").Int(), img.Get("height").Int(), nil
}

func (w *wasmWindow) DrawImageFile(path string, x, y int) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}
	w.ctx.Call("drawImage", img, x, y)
	return nil
}

func (w *wasmWindow) DrawImageFileTo(path string, x, y, width, height, rot int) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	w.ctx.Call("save")

	w.ctx.Call("translate", x+width/2, y+height/2)
	w.ctx.Call("rotate", float64(rot)*math.Pi/180)

	scaleX, scaleY := 1, 1
	if width < 0 {
		scaleX = -1
	}
	if height < 0 {
		scaleY = -1
	}
	if scaleX != 1 || scaleY != 1 {
		w.ctx.Call("scale", scaleX, scaleY)
	}

	w.ctx.Call("drawImage", img,
		0, 0, img.Get("width").Int(), img.Get("height").Int(),
		-width/2, -height/2, width, height,
	)

	w.ctx.Call("restore")
	return nil
}

func (w *wasmWindow) DrawImageFileRotated(path string, x, y, rot int) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	w2 := img.Get("width").Int()
	h2 := img.Get("height").Int()

	w.ctx.Call("save")
	w.ctx.Call("translate", x+w2/2, y+h2/2)
	w.ctx.Call("rotate", float64(rot)*math.Pi/180)
	w.ctx.Call("drawImage", img, -w2/2, -h2/2)
	w.ctx.Call("restore")
	return nil
}

func (w *wasmWindow) DrawImageFilePart(path string,
	sx, sy, sw, sh, dx, dy, dw, dh, rot int,
) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	w.ctx.Call("save")
	w.ctx.Call("translate", dx+dw/2, dy+dh/2)
	w.ctx.Call("rotate", float64(rot)*math.Pi/180)

	scaleX, scaleY := 1, 1
	if dw < 0 {
		scaleX = -1
	}
	if dh < 0 {
		scaleY = -1
	}
	if scaleX != 1 || scaleY != 1 {
		w.ctx.Call("scale", scaleX, scaleY)
	}

	w.ctx.Call("drawImage",
		img,
		sx, sy, sw, sh,
		-dw/2, -dh/2, dw, dh,
	)
	w.ctx.Call("restore")
	return nil
}

func (w *wasmWindow) BlurImages(blur bool) {
	w.ctx.Set("imageSmoothingEnabled", blur)
}

func (w *wasmWindow) GetTextSize(text string) (int, int) {
	return w.GetScaledTextSize(text, 1.0)
}

const (
	wasmFontBaseScale = 13.5
	fontLineGapScale  = 1.24
)

func (w *wasmWindow) GetScaledTextSize(text string, scale float32) (wOut, hOut int) {
	if scale <= 0 {
		return 0, 0
	}

	fontSize := wasmFontBaseScale * float64(scale)
	w.ctx.Set("font", fmt.Sprintf("%.2fpx _draw_font_", fontSize))

	lines := strings.Split(text, "\n")
	maxWidth := 0

	for _, line := range lines {
		width := w.ctx.Call("measureText", line).Get("width").Int()
		if width > maxWidth {
			maxWidth = width
		}
	}

	lineHeight := fontSize * fontLineGapScale
	return maxWidth, int(lineHeight*float64(len(lines)) + 0.9999)
}

func (w *wasmWindow) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1.0, color)
}

func (w *wasmWindow) DrawScaledText(text string, x, y int, scale float32, color Color) {
	if scale <= 0 {
		return
	}

	w.setColor(color)

	fontSize := wasmFontBaseScale * float64(scale)
	w.ctx.Set("font", fmt.Sprintf("%.2fpx _draw_font_", fontSize))

	lineHeight := fontSize * fontLineGapScale

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		w.ctx.Call("fillText", line, x, fontSize+float64(y)+float64(i)*lineHeight)
	}
}

func (w *wasmWindow) PlaySoundFile(path string) error {
	if buffer, ok := w.audioBuffers[path]; ok {
		return w.playBuffer(buffer)
	}

	if OpenFile != nil {
		url, err := loadBlob(path)
		if err != nil {
			return err
		}
		w.loadAndPlaySound(path, url)
	} else {
		w.loadAndPlaySound(path, path)
	}

	return nil
}

func (w *wasmWindow) playBuffer(buffer js.Value) error {
	source := w.audioCtx.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Call("connect", w.audioCtx.Get("destination"))

	// If sound has already started (after first user input) we play the sound
	// right away.
	// If sound is still disabled (until first user input) we remember the
	// sound to be played later.
	if w.hasSeenUserInput {
		source.Call("start")
	} else {
		w.soundsToPlay = append(w.soundsToPlay, futureSound{
			source:    source,
			startedAt: time.Now(),
		})
	}

	return nil
}

func (w *wasmWindow) loadAndPlaySound(path string, url interface{}) {
	w.asyncLoadSound(path, url, func(buffer js.Value, err error) {
		if err == nil {
			w.playBuffer(buffer)
		}
	})
}

func (w *wasmWindow) asyncLoadSound(path string, url interface{}, callback func(js.Value, error)) {
	fetchPromise := js.Global().Call("fetch", url)
	fetchPromise.Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resp := args[0]
		resp.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			arrayBuffer := args[0]
			w.audioCtx.Call("decodeAudioData", arrayBuffer,
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					buffer := args[0]
					w.audioBuffers[path] = buffer
					callback(buffer, nil)
					return nil
				}),
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					callback(js.Null(), fmt.Errorf("failed to decode audio: %s", path))
					return nil
				}),
			)
			return nil
		}))
		return nil
	}))
}
