//go:build js && wasm
// +build js,wasm

package draw

import (
	"fmt"
	"strings"
	"syscall/js"
)

type wasmWindow struct {
	update         UpdateFunction
	canvas         js.Value
	ctx            js.Value
	width, height  int
	running        bool
	keyDown        map[Key]bool
	pressedKeys    []Key
	typedChars     []rune
	mouseX, mouseY int
	mouseDown      map[MouseButton]bool
	wheelX         float64
	wheelY         float64
	clicks         []MouseClick
	imageCache     map[string]js.Value
	audioCtx       js.Value
	audioBuffers   map[string]js.Value
}

func RunWindow(title string, width, height int, update UpdateFunction) error {
	doc := js.Global().Get("document")
	canvas := doc.Call("getElementById", "gameCanvas")
	if !canvas.Truthy() {
		return js.Error{Value: js.ValueOf("canvas element not found")}
	}
	canvas.Set("width", width)
	canvas.Set("height", height)

	ctx := canvas.Call("getContext", "2d")

	win := &wasmWindow{
		update:       update,
		canvas:       canvas,
		ctx:          ctx,
		width:        width,
		height:       height,
		running:      true,
		keyDown:      make(map[Key]bool),
		mouseDown:    make(map[MouseButton]bool),
		imageCache:   make(map[string]js.Value),
		audioCtx:     js.Global().Get("AudioContext").New(),
		audioBuffers: make(map[string]js.Value),
	}

	js.Global().Call("addEventListener", "keydown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		code := event.Get("code").String()
		key := toKey(code)
		if key != 0 {
			if !win.keyDown[key] {
				win.pressedKeys = append(win.pressedKeys, key)
			}
			win.keyDown[key] = true
		}
		return nil
	}))

	js.Global().Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		code := event.Get("code").String()
		key := toKey(code)
		if key != 0 {
			win.keyDown[key] = false
		}
		return nil
	}))

	js.Global().Call("addEventListener", "keyup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		code := event.Get("code").String()
		key := toKey(code)
		if key != 0 {
			win.keyDown[key] = false
		}
		return nil
	}))

	js.Global().Call("addEventListener", "keypress", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		char := rune(event.Get("key").String()[0])
		win.typedChars = append(win.typedChars, char)
		return nil
	}))

	canvas.Call("addEventListener", "mousemove", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		bounds := canvas.Call("getBoundingClientRect")
		wX := event.Get("clientX").Int() - bounds.Get("left").Int()
		wY := event.Get("clientY").Int() - bounds.Get("top").Int()
		win.mouseX = wX
		win.mouseY = wY
		return nil
	}))

	canvas.Call("addEventListener", "mousedown", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		button := event.Get("button").Int()
		win.mouseDown[MouseButton(button)] = true
		win.clicks = append(win.clicks, MouseClick{
			X:      win.mouseX,
			Y:      win.mouseY,
			Button: MouseButton(button),
		})
		return nil
	}))

	canvas.Call("addEventListener", "mouseup", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		button := event.Get("button").Int()
		win.mouseDown[MouseButton(button)] = false
		return nil
	}))

	canvas.Call("addEventListener", "wheel", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		event := args[0]
		deltaX := event.Get("deltaX").Float()
		deltaY := event.Get("deltaY").Float()

		// Normalize direction (scrolling "up" is usually negative)
		win.wheelX += deltaX
		win.wheelY += deltaY

		// Prevent page from scrolling
		event.Call("preventDefault")
		return nil
	}))

	//NOTE: We need to create it with: <canvas id="gameCanvas" width="800" height="600" style="touch-action:none;"></canvas>

	// Call update loop using requestAnimationFrame
	var renderFrame js.Func
	renderFrame = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if win.running {
			win.update(win)

			clicks := win.clicks
			win.clicks = nil // reset for next frame
			win.clicks = append(win.clicks[:0], clicks...)
			win.wheelX = 0
			win.wheelY = 0
			win.pressedKeys = nil
			win.typedChars = nil

			js.Global().Call("requestAnimationFrame", renderFrame)
		}
		return nil
	})
	js.Global().Call("requestAnimationFrame", renderFrame)

	// Prevent main from exiting
	select {}
}

func MathPi() float64 {
	return js.Global().Get("Math").Get("PI").Float()
}

func (w *wasmWindow) setColor(c Color) {
	r := int(c.R * 255)
	g := int(c.G * 255)
	b := int(c.B * 255)
	a := c.A
	w.ctx.Set("fillStyle", fmt.Sprintf("rgba(%d,%d,%d,%f)", r, g, b, a))
	w.ctx.Set("strokeStyle", fmt.Sprintf("rgba(%d,%d,%d,%f)", r, g, b, a))
}

func (w *wasmWindow) loadImage(path string) (js.Value, error) {
	if img, ok := w.imageCache[path]; ok {
		return img, nil
	}

	done := make(chan struct{})
	var img js.Value = js.Global().Get("Image").New()
	var err error

	onLoad := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		w.imageCache[path] = img
		close(done)
		return nil
	})
	onError := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		err = fmt.Errorf("failed to load image: %s", path)
		close(done)
		return nil
	})

	img.Set("onload", onLoad)
	img.Set("onerror", onError)
	img.Set("src", path)

	<-done
	return img, err
}

func (w *wasmWindow) loadSoundFile(path string) (js.Value, error) {
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
			w.audioCtx.Call("decodeAudioData", arrayBuffer,
				js.FuncOf(func(this js.Value, args []js.Value) interface{} {
					result = args[0]
					w.audioBuffers[path] = result
					close(done)
					return nil
				}),
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

func toKey(code string) Key {
	switch code {
	case "KeyA":
		return KeyA
	case "KeyB":
		return KeyB
	case "KeyC":
		return KeyC
	case "KeyD":
		return KeyD
	case "KeyE":
		return KeyE
	case "KeyF":
		return KeyF
	case "KeyG":
		return KeyG
	case "KeyH":
		return KeyH
	case "KeyI":
		return KeyI
	case "KeyJ":
		return KeyJ
	case "KeyK":
		return KeyK
	case "KeyL":
		return KeyL
	case "KeyM":
		return KeyM
	case "KeyN":
		return KeyN
	case "KeyO":
		return KeyO
	case "KeyP":
		return KeyP
	case "KeyQ":
		return KeyQ
	case "KeyR":
		return KeyR
	case "KeyS":
		return KeyS
	case "KeyT":
		return KeyT
	case "KeyU":
		return KeyU
	case "KeyV":
		return KeyV
	case "KeyW":
		return KeyW
	case "KeyX":
		return KeyX
	case "KeyY":
		return KeyY
	case "KeyZ":
		return KeyZ
	case "ArrowLeft":
		return KeyLeft
	case "ArrowRight":
		return KeyRight
	case "ArrowUp":
		return KeyUp
	case "ArrowDown":
		return KeyDown
	case "Enter":
		return KeyEnter
	case "Space":
		return KeySpace
	case "Escape":
		return KeyEscape
	case "Backspace":
		return KeyBackspace
	case "Delete":
		return KeyDelete
	case "Insert":
		return KeyInsert
	case "Home":
		return KeyHome
	case "End":
		return KeyEnd
	case "PageUp":
		return KeyPageUp
	case "PageDown":
		return KeyPageDown
	case "ShiftLeft":
		return KeyLeftShift
	case "ShiftRight":
		return KeyRightShift
	case "ControlLeft":
		return KeyLeftControl
	case "ControlRight":
		return KeyRightControl
	case "AltLeft":
		return KeyLeftAlt
	case "AltRight":
		return KeyRightAlt
	case "Tab":
		return KeyTab
	case "CapsLock":
		return KeyCapslock
	case "NumEnter":
		return KeyNumEnter
	case "NumPlus":
		return KeyNumAdd
	case "NumMinus":
		return KeyNumSubtract
	case "NumMultiply":
		return KeyNumMultiply
	case "NumDivide":
		return KeyNumDivide
	case "Num0":
		return KeyNum0
	case "Num1":
		return KeyNum1
	case "Num2":
		return KeyNum2
	case "Num3":
		return KeyNum3
	case "Num4":
		return KeyNum4
	case "Num5":
		return KeyNum5
	case "Num6":
		return KeyNum6
	case "Num7":
		return KeyNum7
	case "Num8":
		return KeyNum8
	case "Num9":
		return KeyNum9
	case "Digit0":
		return Key0
	case "Digit1":
		return Key1
	case "Digit2":
		return Key2
	case "Digit3":
		return Key3
	case "Digit4":
		return Key4
	case "Digit5":
		return Key5
	case "Digit6":
		return Key6
	case "Digit7":
		return Key7
	case "Digit8":
		return Key8
	case "Digit9":
		return Key9
	case "KeyF1":
		return KeyF1
	case "KeyF2":
		return KeyF2
	case "KeyF3":
		return KeyF3
	case "KeyF4":
		return KeyF4
	case "KeyF5":
		return KeyF5
	case "KeyF6":
		return KeyF6
	case "KeyF7":
		return KeyF7
	case "KeyF8":
		return KeyF8
	case "KeyF9":
		return KeyF9
	case "KeyF10":
		return KeyF10
	case "KeyF11":
		return KeyF11
	case "KeyF12":
		return KeyF12
	}
	return 0
}

func (w *wasmWindow) Close() {
	w.running = false
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
	return string(w.typedChars)
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
	w.FillRect(x, y, 1, 1, c)
}

func (w *wasmWindow) DrawLine(x1, y1, x2, y2 int, c Color) {
	w.setColor(c)
	w.ctx.Call("beginPath")
	w.ctx.Call("moveTo", x1, y1)
	w.ctx.Call("lineTo", x2, y2)
	w.ctx.Call("stroke")
}

func (w *wasmWindow) DrawRect(x, y, width, height int, c Color) {
	w.setColor(c)
	w.ctx.Call("strokeRect", x, y, width, height)
}

func (w *wasmWindow) FillRect(x, y, width, height int, c Color) {
	w.setColor(c)
	w.ctx.Call("fillRect", x, y, width, height)
}

func (w *wasmWindow) DrawEllipse(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}
	w.setColor(color)
	w.ctx.Call("beginPath")
	w.ctx.Call("ellipse",
		x+width/2,  // centerX
		y+height/2, // centerY
		width/2,    // radiusX
		height/2,   // radiusY
		0,          // rotation in radians
		0,          // startAngle
		2*MathPi(), // endAngle
	)
	w.ctx.Call("stroke")
}

func (w *wasmWindow) FillEllipse(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}
	w.setColor(color)
	w.ctx.Call("beginPath")
	w.ctx.Call("ellipse",
		x+width/2,
		y+height/2,
		width/2,
		height/2,
		0,
		0,
		2*MathPi(),
	)
	w.ctx.Call("fill")
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

func (w *wasmWindow) DrawImageFileTo(path string, x, y, w2, h2, rot int) error {
	img, err := w.loadImage(path)
	if err != nil {
		return err
	}

	// Save current context
	w.ctx.Call("save")

	// Translate to center of target rect
	w.ctx.Call("translate", x+w2/2, y+h2/2)
	w.ctx.Call("rotate", float64(rot)*MathPi()/180)

	// Draw centered image
	w.ctx.Call("drawImage", img,
		0, 0, img.Get("width").Int(), img.Get("height").Int(), // source
		-w2/2, -h2/2, w2, h2, // destination (centered)
	)

	// Restore context
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
	w.ctx.Call("rotate", float64(rot)*MathPi()/180)
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
	w.ctx.Call("rotate", float64(rot)*MathPi()/180)
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
	w.ctx.Set("imageSmoothingEnabled", blur)
}

func (w *wasmWindow) GetTextSize(text string) (int, int) {
	return w.GetScaledTextSize(text, 1.0)
}

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

func (w *wasmWindow) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1.0, color)
}

func (w *wasmWindow) DrawScaledText(text string, x, y int, scale float32, color Color) {
	if scale <= 0 {
		return
	}

	w.setColor(color)
	fontSize := 16.0 * float64(scale) // base size of 16, feel free to tweak
	w.ctx.Set("font", fmt.Sprintf("%.2fpx monospace", fontSize))
	lines := strings.Split(text, "\n")
	lineHeight := int(fontSize * 1.2) // line spacing

	for i, line := range lines {
		w.ctx.Call("fillText", line, x, y+i*lineHeight)
	}
}

func (w *wasmWindow) PlaySoundFile(path string) error {
	buffer, err := w.loadSoundFile(path)
	if err != nil {
		return err
	}

	source := w.audioCtx.Call("createBufferSource")
	source.Set("buffer", buffer)
	source.Call("connect", w.audioCtx.Call("destination"))
	source.Call("start")
	return nil
}
