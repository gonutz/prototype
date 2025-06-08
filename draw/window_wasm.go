//go:build js && wasm
// +build js,wasm

package draw

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"syscall/js"
)

type wasmWindow struct {
	update          UpdateFunction
	canvas          js.Value
	ctx             js.Value
	width, height   int
	running         bool
	keyDown         [keyCount]bool
	pressedKeys     []Key
	typedChars      []rune
	mouseX, mouseY  int
	mouseDown       [mouseButtonCount]bool
	wheelX          float64
	wheelY          float64
	clicks          []MouseClick
	imageCache      map[string]js.Value
	imagesLoaded    chan struct{}
	pendingImages   map[string]bool
	audioCtx        js.Value
	audioBuffers    map[string]js.Value
	closeImagesOnce sync.Once
}

func bindEvent(target js.Value, event string, handler func(js.Value)) js.Func {
	jsFunc := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		handler(args[0])
		return nil
	})
	target.Call("addEventListener", event, jsFunc)
	return jsFunc
}

// RunWindow initializes a WebAssembly window with an HTML canvas element, sets
// up input and rendering, and starts the main update loop.
func RunWindow(title string, width, height int, update UpdateFunction) error {
	doc := js.Global().Get("document")
	doc.Set("title", title)
	canvas := doc.Call("getElementById", "gameCanvas")
	if !canvas.Truthy() {
		return js.Error{Value: js.ValueOf("canvas element not found")}
	}
	canvas.Set("width", width)
	canvas.Set("height", height)

	ctx := canvas.Call("getContext", "2d")

	// Create the wasmWindow instance with input states, rendering context, and audio
	win := &wasmWindow{
		update:       update,
		canvas:       canvas,
		ctx:          ctx,
		width:        width,
		height:       height,
		running:      true,
		imageCache:   make(map[string]js.Value),
		audioCtx:     js.Global().Get("AudioContext").New(),
		audioBuffers: make(map[string]js.Value),
	}

	win.pendingImages = make(map[string]bool)
	win.imagesLoaded = make(chan struct{})

	// Handles key press events: resumes audio and tracks pressed keys.
	bindEvent(js.Global(), "keydown", func(e js.Value) {
		keyCode := e.Get("code").String()
		keyValue := e.Get("key").String()
		key := toKey(keyCode, keyValue)

		if win.audioCtx.Get("state").String() == "suspended" {
			win.audioCtx.Call("resume")
		}

		if key != 0 && !win.keyDown[key] {
			win.pressedKeys = append(win.pressedKeys, key)
		}
		win.keyDown[key] = true

		e.Call("preventDefault")
	})

	// Handles key release events
	bindEvent(js.Global(), "keyup", func(e js.Value) {
		keyCode := e.Get("code").String()
		keyValue := e.Get("key").String()
		key := toKey(keyCode, keyValue)
		if key != 0 {
			win.keyDown[key] = false
		}
		e.Call("preventDefault")
	})

	// Character input (text entry)
	bindEvent(js.Global(), "keypress", func(e js.Value) {
		keyStr := e.Get("key").String()
		if len(keyStr) > 0 {
			win.typedChars = append(win.typedChars, rune(keyStr[0]))
		}
	})

	// Mouse movement tracking
	bindEvent(doc, "mousemove", func(e js.Value) {
		bounds := canvas.Call("getBoundingClientRect")
		win.mouseX = e.Get("clientX").Int() - bounds.Get("left").Int()
		win.mouseY = e.Get("clientY").Int() - bounds.Get("top").Int()
	})

	// To determine whether the mouse buttons are currently up or down, we
	// register the mouse down and up events on the document.
	// To collect mouse clicks, we register the mouse down event on the canvas.
	// Clicks outside the canvas are not reported.
	bindEvent(doc, "mousedown", func(e js.Value) {
		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			win.mouseDown[button] = true
		}
	})
	bindEvent(doc, "mouseup", func(e js.Value) {
		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			win.mouseDown[button] = false
		}
	})
	bindEvent(canvas, "mousedown", func(e js.Value) {
		button := e.Get("button").Int()
		if 0 <= button && button < int(mouseButtonCount) {
			win.clicks = append(win.clicks, MouseClick{
				X:      win.mouseX,
				Y:      win.mouseY,
				Button: MouseButton(button),
			})
		}
	})

	// Mouse wheel
	bindEvent(canvas, "wheel", func(e js.Value) {
		win.wheelX -= e.Get("deltaX").Float() / 100
		win.wheelY -= e.Get("deltaY").Float() / 100
		e.Call("preventDefault") // prevent page scroll
	})

	// Suppress right clicks triggering the context menu.
	bindEvent(canvas, "contextmenu", func(e js.Value) {
		e.Call("preventDefault")
	})

	// Main render loop using requestAnimationFrame
	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		win.FillRect(0, 0, win.width, win.height, Black)
		if win.running {
			win.update(win)
			// Reset input state between frames.
			win.wheelX = 0
			win.wheelY = 0
			win.clicks = win.clicks[:0]
			win.pressedKeys = win.pressedKeys[:0]
			win.typedChars = win.typedChars[:0]
		}
		js.Global().Call("requestAnimationFrame", renderFrame)
		return nil
	})
	js.Global().Call("requestAnimationFrame", renderFrame)

	// Prevent Go main from exiting (WASM requires this to keep running)
	select {}
}

// setColor sets both fill and stroke styles on the canvas context
// based on the provided RGBA color. Each color component is converted
// to its 0–255 representation for use with CSS-style RGBA strings.
func (w *wasmWindow) setColor(c Color) {
	r := int(c.R * 255)
	g := int(c.G * 255)
	b := int(c.B * 255)
	a := c.A
	w.ctx.Set("fillStyle", fmt.Sprintf("rgba(%d,%d,%d,%f)", r, g, b, a))
	w.ctx.Set("strokeStyle", fmt.Sprintf("rgba(%d,%d,%d,%f)", r, g, b, a))
}

// loadImage loads an image from the given path and returns the corresponding
// JavaScript image element. The result is cached to avoid redundant network requests.
//
// The function sets up onload and onerror callbacks to resolve a Go channel
// once the image is successfully loaded or has failed to load.
func (w *wasmWindow) loadImage(path string) (js.Value, error) {
	if img, ok := w.imageCache[path]; ok && img.Truthy() {
		return img, nil
	}

	if _, loading := w.pendingImages[path]; loading {
		return js.Null(), fmt.Errorf("image still loading: %s", path)
	}

	w.pendingImages[path] = true

	img := js.Global().Get("Image").New()

	var onLoadFunc, onErrorFunc js.Func

	cleanup := func() {
		delete(w.pendingImages, path)
		if len(w.pendingImages) == 0 {
			w.closeImagesOnce.Do(func() { close(w.imagesLoaded) })
		}
	}

	// Allocate and bind onload handler
	onLoadFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		onLoadFunc.Release()
		onErrorFunc.Release()

		w.imageCache[path] = img
		cleanup()
		return nil
	})

	// Allocate and bind onerror handler
	onErrorFunc = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		onLoadFunc.Release()
		onErrorFunc.Release()

		cleanup()
		return nil
	})

	img.Set("onload", onLoadFunc)
	img.Set("onerror", onErrorFunc)
	img.Set("src", path)

	return js.Null(), fmt.Errorf("image still loading: %s", path)
}

// loadSoundFile fetches and decodes an audio file from the given path using the Web Audio API.
// It returns a decoded AudioBuffer that can be played via PlaySoundFile.
//
// The result is cached in audioBuffers to avoid redundant decoding on repeated calls.
// This function blocks using a channel until the asynchronous JS fetch and decode are complete.
func (w *wasmWindow) loadSoundFile(path string) (js.Value, error) {
	// Return cached buffer if already loaded
	if buffer, ok := w.audioBuffers[path]; ok {
		return buffer, nil
	}

	done := make(chan struct{})
	var result js.Value
	var err error

	fetchPromise := js.Global().Call("fetch", path)
	then := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		resp := args[0]
		resp.Call("arrayBuffer").Call("then", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			arrayBuffer := args[0]

			// Decode the ArrayBuffer into an AudioBuffer using decodeAudioData
			w.audioCtx.Call("decodeAudioData", arrayBuffer,
				// Success callback
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					result = args[0]
					w.audioBuffers[path] = result
					close(done)
					return nil
				}),
				// Error callback
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					err = fmt.Errorf("failed to decode audio: %s", path)
					close(done)
					return nil
				}),
			)
			return nil
		}))
		return nil
	})

	fetchPromise.Call("then", then)
	<-done

	return result, err
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
	"KeyF1":          KeyF1,
	"KeyF2":          KeyF2,
	"KeyF3":          KeyF3,
	"KeyF4":          KeyF4,
	"KeyF5":          KeyF5,
	"KeyF6":          KeyF6,
	"KeyF7":          KeyF7,
	"KeyF8":          KeyF8,
	"KeyF9":          KeyF9,
	"KeyF10":         KeyF10,
	"KeyF11":         KeyF11,
	"KeyF12":         KeyF12,
	"KeyF13":         KeyF13,
	"KeyF14":         KeyF14,
	"KeyF15":         KeyF15,
	"KeyF16":         KeyF16,
	"KeyF17":         KeyF17,
	"KeyF18":         KeyF18,
	"KeyF19":         KeyF19,
	"KeyF20":         KeyF20,
	"KeyF21":         KeyF21,
	"KeyF22":         KeyF22,
	"KeyF23":         KeyF23,
	"KeyF24":         KeyF24,
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
	// TODO Stop all sounds.
}

func (w *wasmWindow) Size() (int, int) {
	return w.width, w.height
}

func (w *wasmWindow) SetFullscreen(f bool) {
	if f {
		w.canvas.Call("requestFullscreen")
	} else {
		doc := js.Global().Get("document")
		if doc.Call("exitFullscreen").Truthy() {
			doc.Call("exitFullscreen")
		}
	}
}

func (w *wasmWindow) ShowCursor(show bool) {
	if show {
		w.canvas.Get("style").Set("cursor", "default")
	} else {
		w.canvas.Get("style").Set("cursor", "none")
	}
}

// WasKeyPressed returns true if the given key was pressed during this frame.
// Use this for single-trigger events (e.g., jumping, opening menus).
func (w *wasmWindow) WasKeyPressed(key Key) bool {
	for _, k := range w.pressedKeys {
		if k == key {
			return true
		}
	}
	return false
}

// IsKeyDown returns true if the given key is currently held down.
// Use this for continuous input (e.g., holding movement keys).
func (w *wasmWindow) IsKeyDown(key Key) bool {
	return w.keyDown[key]
}

// Characters returns a string of characters typed by the user during this frame.
// Useful for text input fields or typing games.
func (w *wasmWindow) Characters() string {
	return string(w.typedChars)
}

// IsMouseDown returns true if the specified mouse button is currently pressed.
func (w *wasmWindow) IsMouseDown(button MouseButton) bool {
	return w.mouseDown[button]
}

// Clicks returns a slice of all mouse clicks registered during this frame.
// Each MouseClick contains the position and button.
// The slice is cleared after each update.
func (w *wasmWindow) Clicks() []MouseClick {
	return w.clicks
}

// MousePosition returns the current mouse cursor position relative to the canvas.
func (w *wasmWindow) MousePosition() (int, int) {
	return w.mouseX, w.mouseY
}

// MouseWheelX returns the accumulated horizontal scroll value for the current frame.
func (w *wasmWindow) MouseWheelX() float64 {
	return w.wheelX
}

// MouseWheelY returns the accumulated vertical scroll value for the current frame.
func (w *wasmWindow) MouseWheelY() float64 {
	return w.wheelY
}

// DrawPoint renders a single pixel (1x1 rectangle) at (x, y) using the specified color.
func (w *wasmWindow) DrawPoint(x, y int, c Color) {
	w.setColor(c)
	w.ctx.Call("fillRect", x, y, 1, 1)
}

// DrawLine renders a straight line between (x1, y1) and (x2, y2) with the given color.
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

// DrawRect outlines a rectangle using stroke style at the given position and size.
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

// FillRect renders a solid filled rectangle.
func (w *wasmWindow) FillRect(x, y, width, height int, c Color) {
	if width <= 0 || height <= 0 {
		return
	}

	w.setColor(c)
	w.ctx.Call("fillRect", x, y, width, height)
}

// DrawEllipse draws an outlined ellipse within the bounding rectangle at (x, y, width, height).
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

// FillEllipse draws a filled ellipse within the bounding rectangle.
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

// ImageSize returns the native width and height of the image at the given path.
// The image is loaded (or retrieved from cache) if needed.
func (w *wasmWindow) ImageSize(path string) (int, int, error) {
	img, err := w.loadImage(path)
	if err != nil {
		return 0, 0, err
	}
	return img.Get("width").Int(), img.Get("height").Int(), nil
}

// DrawImageFile draws an image at the given position.
// If the image is not loaded yet, nothing is drawn.
func (w *wasmWindow) DrawImageFile(path string, x, y int) error {
	img, err := w.loadImage(path)
	if err != nil || !img.Truthy() {
		return nil
	}
	w.ctx.Call("drawImage", img, x, y)
	return nil
}

// DrawImageFileTo draws an image scaled to a new size and rotated (in degrees) around its center.
func (w *wasmWindow) DrawImageFileTo(path string, x, y, w2, h2, rot int) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	// Save current context
	w.ctx.Call("save")

	// Translate to center of target rect
	w.ctx.Call("translate", x+w2/2, y+h2/2)
	w.ctx.Call("rotate", float64(rot)*math.Pi/180)

	// Draw centered image
	w.ctx.Call("drawImage", img,
		0, 0, img.Get("width").Int(), img.Get("height").Int(), // source
		-w2/2, -h2/2, w2, h2, // destination (centered)
	)

	// Restore context
	w.ctx.Call("restore")
	return nil
}

// DrawImageFileRotated draws the image at (x, y), rotated by `rot` degrees about its center.
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

// DrawImageFilePart draws a subsection of the image, defined by source rect (sx, sy, sw, sh),
// to a destination rect (dx, dy, dw, dh) and applies rotation (degrees) around its center.
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
	w.ctx.Call("drawImage",
		img,
		sx, sy, sw, sh, // source rect
		-dw/2, -dh/2, dw, dh, // destination rect, centered
	)
	w.ctx.Call("restore")
	return nil
}

func (w *wasmWindow) BlurImages(blur bool) {
	w.ctx.Set("imageSmoothingEnabled", blur)
}

func (w *wasmWindow) BlurText(blur bool) {
	// TODO Figure out how we want to draw and blur text.
}

// GetTextSize returns the width and height (in pixels) required to render the given text at default scale.
func (w *wasmWindow) GetTextSize(text string) (int, int) {
	return w.GetScaledTextSize(text, 1.0)
}

// GetScaledTextSize returns the pixel dimensions required to render text at the given scale.
// Line breaks are taken into account.
func (w *wasmWindow) GetScaledTextSize(text string, scale float32) (wOut, hOut int) {
	if scale <= 0 {
		return 0, 0
	}

	fontSize := 16.0 * float64(scale)
	w.ctx.Set("font", fmt.Sprintf("%.2fpx monospace", fontSize))
	lines := strings.Split(text, "\n")
	maxWidth := 0

	for _, line := range lines {
		width := w.ctx.Call("measureText", line).Get("width").Int()
		if width > maxWidth {
			maxWidth = width
		}
	}

	lineHeight := int(fontSize * 1.2)
	return maxWidth, lineHeight * len(lines)
}

// DrawText renders a string at (x, y) using the given color and default scale (1.0).
func (w *wasmWindow) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1.0, color)
}

// DrawScaledText renders a string of text at the given position with a scaling factor and color.
// Text is drawn using a monospace font, and supports multi-line input (lines split by '\n').
func (w *wasmWindow) DrawScaledText(text string, x, y int, scale float32, color Color) {
	// Ignore zero or negative scale
	if scale <= 0 {
		return
	}

	// Set fill color for the text
	w.setColor(color)

	// Compute font size based on scaling factor
	fontSize := 16.0 * float64(scale) // base size of 16, feel free to tweak

	// Apply font style to canvas context (monospace font for uniform spacing)
	w.ctx.Set("font", fmt.Sprintf("%.2fpx monospace", fontSize))

	// Split the input into lines
	lines := strings.Split(text, "\n")

	// Define line spacing as 1.2x font size
	lineHeight := int(fontSize * 1.2) // line spacing

	// Draw each line at its vertical offset
	for i, line := range lines {
		w.ctx.Call("fillText", line, x, y+i*lineHeight)
	}
}

// PlaySoundFile plays an audio file by path using the Web Audio API.
// It ensures the AudioContext is resumed before playback, as required by browser policies.
func (w *wasmWindow) PlaySoundFile(path string) error {
	// Do not wait on resume or fetch — just try it
	if w.audioCtx.Get("state").String() == "suspended" {
		w.audioCtx.Call("resume")
	}

	// Already loaded? Play immediately
	if buffer, ok := w.audioBuffers[path]; ok {
		return w.playBuffer(buffer)
	}

	// Begin async load
	w.asyncLoadSound(path, func(buffer js.Value, err error) {
		if err == nil {
			w.playBuffer(buffer)
		}
	})

	return nil
}

// Non-blocking async sound load using JS promises
func (w *wasmWindow) asyncLoadSound(path string, callback func(js.Value, error)) {
	fetchPromise := js.Global().Call("fetch", path)
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

// Small helper to play a buffer
func (w *wasmWindow) playBuffer(buffer js.Value) error {
	source := w.audioCtx.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Call("connect", w.audioCtx.Get("destination"))
	source.Call("start")
	return nil
}
