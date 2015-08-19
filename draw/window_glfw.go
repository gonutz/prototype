// +build glfw !sdl2

package draw

import (
	"bytes"
	"errors"
	"github.com/go-gl/gl/v2.1/gl"
	"github.com/go-gl/glfw/v3.1/glfw"
	"image"
	"image/draw"
	_ "image/png"
	"io"
	"math"
	"os"
	"runtime"
	"strings"
	"time"
)

type window struct {
	running        bool
	pressed        []string
	typed          []rune
	window         *glfw.Window
	width, height  float64
	textures       map[string]texture
	clicks         []MouseClick
	mouseX, mouseY int
}

func RunWindow(title string, width, height int, flags int, update UpdateFunction) error {
	err := glfw.Init()
	if err != nil {
		return err
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.ContextVersionMajor, 1)
	glfw.WindowHint(glfw.ContextVersionMinor, 0)
	if flags&Resizable > 0 {
		glfw.WindowHint(glfw.Resizable, glfw.True)
	} else {
		glfw.WindowHint(glfw.Resizable, glfw.False)
	}

	win, err := glfw.CreateWindow(width, height, title, nil, nil)
	if err != nil {
		return err
	}
	win.MakeContextCurrent()
	// center the window on the screen (omitting the window border)
	screen := glfw.GetMonitors()[0].GetVideoMode()
	win.SetPos((screen.Width-width)/2, (screen.Height-height)/2)

	err = gl.Init()
	if err != nil {
		return err
	}
	gl.MatrixMode(gl.PROJECTION)
	gl.Ortho(0, float64(width), float64(height), 0, -1, 1)
	gl.MatrixMode(gl.MODELVIEW)
	gl.Enable(gl.BLEND)
	gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)

	w := &window{
		running:  true,
		window:   win,
		width:    float64(width),
		height:   float64(height),
		textures: make(map[string]texture),
	}
	win.SetKeyCallback(w.keyPress)
	win.SetCharCallback(w.charTyped)
	win.SetMouseButtonCallback(w.mouseButtonEvent)
	win.SetCursorPosCallback(w.mousePositionChanged)
	win.SetSizeCallback(func(_ *glfw.Window, width, height int) {
		w.width, w.height = float64(width), float64(height)
		gl.MatrixMode(gl.PROJECTION)
		gl.LoadIdentity()
		gl.Ortho(0, w.width, w.height, 0, -1, 1)
		gl.Viewport(0, 0, int32(width), int32(height))
		gl.MatrixMode(gl.MODELVIEW)
	})

	lastUpdateTime := time.Now().Add(-time.Hour)
	const updateInterval = 1.0 / 60.0
	for w.running && !win.ShouldClose() {
		glfw.PollEvents()

		now := time.Now()
		if now.Sub(lastUpdateTime).Seconds() > updateInterval {
			gl.ClearColor(0, 0, 0, 1)
			gl.Clear(gl.COLOR_BUFFER_BIT)
			update(w)

			w.pressed = nil
			w.typed = nil
			w.clicks = nil

			lastUpdateTime = now
			win.SwapBuffers()
		} else {
			time.Sleep(time.Millisecond)
		}
	}

	w.cleanUp()

	return nil
}

func (w *window) Close() {
	w.running = false
}

func (c *window) Size() (int, int) {
	return int(c.width + 0.5), int(c.height + 0.5)
}

func (w *window) keyPress(_ *glfw.Window, key glfw.Key, _ int, action glfw.Action, _ glfw.ModifierKey) {
	if action == glfw.Press || action == glfw.Repeat {
		w.pressed = append(w.pressed, keyToString[key])
	}
}

func (w *window) WasKeyPressed(key string) bool {
	key = strings.ToLower(key)
	for _, pressed := range w.pressed {
		if pressed == key {
			return true
		}
	}
	return false
}

func (w *window) WasCharTyped(char rune) bool {
	for _, typed := range w.typed {
		if char == typed {
			return true
		}
	}
	return false
}

func (w *window) charTyped(_ *glfw.Window, char rune) {
	w.typed = append(w.typed, char)
}

func (w *window) IsKeyDown(key string) bool {
	key = strings.ToLower(key)
	return w.window.GetKey(stringToKey[key]) == glfw.Press
}

func (w *window) DrawPoint(x, y int, color Color) {
	gl.Begin(gl.POINTS)

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y))

	gl.End()
}

func (w *window) FillRect(x, y, width, height int, color Color) {
	gl.Begin(gl.QUADS)
	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x+width), int32(y))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x+width), int32(y+height))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y+height))
	gl.End()
}

func (w *window) DrawRect(x, y, width, height int, color Color) {
	gl.Begin(gl.LINE_STRIP)

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x+width), int32(y))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x+width), int32(y+height))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y+height))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y-1))

	gl.End()
}

func (w *window) DrawLine(x, y, x2, y2 int, color Color) {
	if x == x2 && y == y2 {
		w.DrawPoint(x, y, color)
		return
	}
	gl.Begin(gl.LINES)

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x), int32(y))

	gl.Color4f(color.R, color.G, color.B, color.A)
	gl.Vertex2i(int32(x2+sign(x2-x)), int32(y2+sign(y2-y)))

	gl.End()
}

func sign(x int) int {
	if x == 0 {
		return 0
	}
	if x > 0 {
		return 1
	}
	return -1
}

var keyToString map[glfw.Key]string
var stringToKey map[string]glfw.Key

type texture struct {
	id   uint32
	w, h int
}

func (w *window) loadTexture(r io.Reader, name string) (texture, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return texture{}, err
	}

	var rgba *image.RGBA
	if asRGBA, ok := img.(*image.RGBA); ok {
		rgba = asRGBA
	} else {
		rgba = image.NewRGBA(img.Bounds())
		if rgba.Stride != rgba.Rect.Size().X*4 {
			return texture{}, errors.New("unsupported stride")
		}
		draw.Draw(rgba, rgba.Bounds(), img, image.Point{0, 0}, draw.Src)
	}

	var tex uint32
	gl.Enable(gl.TEXTURE_2D)
	gl.GenTextures(1, &tex)
	gl.BindTexture(gl.TEXTURE_2D, tex)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.CLAMP_TO_EDGE)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.CLAMP_TO_EDGE)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA,
		int32(rgba.Bounds().Dx()),
		int32(rgba.Bounds().Dy()),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(rgba.Pix),
	)
	gl.Disable(gl.TEXTURE_2D)

	w.textures[name] = texture{
		id: tex,
		w:  rgba.Bounds().Dx(),
		h:  rgba.Bounds().Dy(),
	}

	return w.textures[name], nil
}

func (w *window) getOrLoadTexture(path string) (texture, error) {
	if tex, ok := w.textures[path]; ok {
		return tex, nil
	}

	imgFile, err := os.Open(path)
	if err != nil {
		return texture{}, err
	}
	defer imgFile.Close()

	return w.loadTexture(imgFile, path)
}

func (w *window) cleanUp() {
	for _, tex := range w.textures {
		gl.DeleteTextures(1, &tex.id)
	}
	w.textures = nil
}

func (w *window) Clicks() []MouseClick {
	return w.clicks
}

func (win *window) mouseButtonEvent(w *glfw.Window, button glfw.MouseButton, action glfw.Action, mod glfw.ModifierKey) {
	if action == glfw.Press {
		b := toMouseButton(button)
		x, y := win.window.GetCursorPos()
		win.clicks = append(win.clicks, MouseClick{X: int(x), Y: int(y), Button: b})
	}
}

func (w *window) mousePositionChanged(_ *glfw.Window, x, y float64) {
	w.mouseX, w.mouseY = int(x+0.5), int(y+0.5)
}

func (w *window) MouseX() int { return w.mouseX }
func (w *window) MouseY() int { return w.mouseY }

func toMouseButton(b glfw.MouseButton) MouseButton {
	if b == glfw.MouseButtonRight {
		return RightButton
	}
	if b == glfw.MouseButtonMiddle {
		return MiddleButton
	}
	return LeftButton
}

func (w *window) IsMouseDown(button MouseButton) bool {
	return w.window.GetMouseButton(toGlfwButton(button)) == glfw.Press
}

func toGlfwButton(b MouseButton) glfw.MouseButton {
	if b == RightButton {
		return glfw.MouseButtonRight
	}
	if b == MiddleButton {
		return glfw.MouseButtonMiddle
	}
	return glfw.MouseButtonLeft
}

func (w *window) DrawEllipse(x, y, width, height int, color Color) {
	w.ellipse(false, x, y, width, height, color)
}

func (w *window) FillEllipse(x, y, width, height int, color Color) {
	w.ellipse(true, x, y, width, height, color)
}

func (w *window) ellipse(filled bool, x, y, width, height int, color Color) {
	a, b := float32(width)/2, float32(height)/2
	fx, fy := float32(x)+a, float32(y)+b

	if filled {
		gl.Begin(gl.TRIANGLE_FAN)
		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.Vertex2f(fx, fy)
	} else {
		gl.Begin(gl.LINE_STRIP)
	}

	const stepCount = 50
	const dAngle = 2 * math.Pi / stepCount
	for i, angle := 0, 0.0; i <= stepCount; i, angle = i+1, angle+dAngle {
		sin, cos := math.Sincos(angle)
		x, y := a*float32(cos), b*float32(sin)
		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.Vertex2f(fx+x, fy+y)
	}

	gl.End()
}

func (w *window) DrawImageFile(path string, x, y int) error {
	tex, err := w.getOrLoadTexture(path)
	if err != nil {
		return err
	}

	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.Begin(gl.QUADS)

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(0, 0)
	gl.Vertex2i(int32(x), int32(y))

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(1, 0)
	gl.Vertex2i(int32(x+tex.w), int32(y))

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(1, 1)
	gl.Vertex2i(int32(x+tex.w), int32(y+tex.h))

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(0, 1)
	gl.Vertex2i(int32(x), int32(y+tex.h))

	gl.End()
	gl.Disable(gl.TEXTURE_2D)

	return nil
}

func (win *window) DrawImageFileTo(path string, x, y, w, h, degrees int) error {
	tex, err := win.getOrLoadTexture(path)
	if err != nil {
		return err
	}

	x1, y1 := float32(x), float32(y)
	x2, y2 := float32(x+w-0), float32(y+h-0)
	cx, cy := x1+float32(w)/2, y1+float32(h)/2
	sin, cos := math.Sincos(float64(degrees) / 180 * math.Pi)
	sin32, cos32 := float32(sin), float32(cos)
	p := [4]point{
		{x1, y1},
		{x2, y1},
		{x2, y2},
		{x1, y2},
	}
	for i := range p {
		p[i].x, p[i].y = p[i].x-cx, p[i].y-cy
		p[i].x, p[i].y = cos32*p[i].x-sin32*p[i].y, sin32*p[i].x+cos32*p[i].y
		p[i].x, p[i].y = p[i].x+cx, p[i].y+cy
	}

	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, tex.id)
	gl.Begin(gl.QUADS)

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(0, 0)
	gl.Vertex2f(p[0].x, p[0].y)

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(1, 0)
	gl.Vertex2f(p[1].x, p[1].y)

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(1, 1)
	gl.Vertex2f(p[2].x, p[2].y)

	gl.Color4f(1, 1, 1, 1)
	gl.TexCoord2i(0, 1)
	gl.Vertex2f(p[3].x, p[3].y)

	gl.End()
	gl.Disable(gl.TEXTURE_2D)

	return nil
}

type point struct{ x, y float32 }

func (win *window) GetTextSize(text string) (w, h int) {
	return win.GetScaledTextSize(text, 1.0)
}

func (win *window) GetScaledTextSize(text string, scale float32) (w, h int) {
	if len(text) == 0 {
		return 0, 0
	}
	fontTexture, ok := win.textures[fontTextureID]
	if !ok {
		return 0, 0
	}
	w = int(float32(fontTexture.w/16)*scale + 0.5)
	h = int(float32(fontTexture.h/16)*scale + 0.5)
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

const fontTextureID = "///font_texture"

func (w *window) DrawScaledText(text string, x, y int, scale float32, color Color) {
	fontTexture, ok := w.textures[fontTextureID]
	if !ok {
		var err error
		fontTexture, err = w.loadTexture(bytes.NewReader(bitmapFontWhitePng[:]), fontTextureID)
		if err != nil {
			panic(err)
		}
	}

	width, height := int32(fontTexture.w/16), int32(fontTexture.h/16)
	width = int32(float32(width)*scale + 0.5)
	height = int32(float32(height)*scale + 0.5)

	var srcX, srcY float32
	destX, destY := int32(x), int32(y)

	gl.Enable(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, fontTexture.id)

	gl.Begin(gl.QUADS)
	for _, char := range []byte(text) {
		if char == '\n' {
			destX = int32(x)
			destY += height
			continue
		}
		srcX = float32(int(char)%16) / 16
		srcY = float32(int(char)/16) / 16

		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.TexCoord2f(srcX, srcY)
		gl.Vertex2i(destX, destY)

		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.TexCoord2f(srcX+1.0/16, srcY)
		gl.Vertex2i(destX+width, destY)

		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.TexCoord2f(srcX+1.0/16, srcY+1.0/16)
		gl.Vertex2i(destX+width, destY+height)

		gl.Color4f(color.R, color.G, color.B, color.A)
		gl.TexCoord2f(srcX, srcY+1.0/16)
		gl.Vertex2i(destX, destY+height)

		destX += width
	}
	gl.End()
	gl.Disable(gl.TEXTURE_2D)
}

func (w *window) PlaySoundFile(path string) error {
	return playSoundFile(path)
}

func init() {
	runtime.LockOSThread()
	initKeyMap()
}

func initKeyMap() {
	keyToString = make(map[glfw.Key]string)
	keyToString[glfw.KeyUnknown] = "UNKNOWN"
	keyToString[glfw.KeyEnter] = "ENTER"
	keyToString[glfw.KeyEscape] = "ESCAPE"
	keyToString[glfw.KeyBackspace] = "BACKSPACE"
	keyToString[glfw.KeyTab] = "TAB"
	keyToString[glfw.KeySpace] = "SPACE"
	keyToString[glfw.KeyComma] = "COMMA"
	keyToString[glfw.KeyMinus] = "MINUS"
	keyToString[glfw.KeyPeriod] = "PERIOD"
	keyToString[glfw.KeySlash] = "SLASH"
	keyToString[glfw.Key0] = "0"
	keyToString[glfw.Key1] = "1"
	keyToString[glfw.Key2] = "2"
	keyToString[glfw.Key3] = "3"
	keyToString[glfw.Key4] = "4"
	keyToString[glfw.Key5] = "5"
	keyToString[glfw.Key6] = "6"
	keyToString[glfw.Key7] = "7"
	keyToString[glfw.Key8] = "8"
	keyToString[glfw.Key9] = "9"
	keyToString[glfw.KeySemicolon] = "SEMICOLON"
	keyToString[glfw.KeyLeftBracket] = "LEFTBRACKET"
	keyToString[glfw.KeyBackslash] = "BACKSLASH"
	keyToString[glfw.KeyRightBracket] = "RIGHTBRACKET"
	keyToString[glfw.KeyA] = "a"
	keyToString[glfw.KeyB] = "b"
	keyToString[glfw.KeyC] = "c"
	keyToString[glfw.KeyD] = "d"
	keyToString[glfw.KeyE] = "e"
	keyToString[glfw.KeyF] = "f"
	keyToString[glfw.KeyG] = "g"
	keyToString[glfw.KeyH] = "h"
	keyToString[glfw.KeyI] = "i"
	keyToString[glfw.KeyJ] = "j"
	keyToString[glfw.KeyK] = "k"
	keyToString[glfw.KeyL] = "l"
	keyToString[glfw.KeyM] = "m"
	keyToString[glfw.KeyN] = "n"
	keyToString[glfw.KeyO] = "o"
	keyToString[glfw.KeyP] = "p"
	keyToString[glfw.KeyQ] = "q"
	keyToString[glfw.KeyR] = "r"
	keyToString[glfw.KeyS] = "s"
	keyToString[glfw.KeyT] = "t"
	keyToString[glfw.KeyU] = "u"
	keyToString[glfw.KeyV] = "v"
	keyToString[glfw.KeyW] = "w"
	keyToString[glfw.KeyX] = "x"
	keyToString[glfw.KeyY] = "y"
	keyToString[glfw.KeyZ] = "z"
	keyToString[glfw.KeyCapsLock] = "CAPSLOCK"
	keyToString[glfw.KeyF1] = "F1"
	keyToString[glfw.KeyF2] = "F2"
	keyToString[glfw.KeyF3] = "F3"
	keyToString[glfw.KeyF4] = "F4"
	keyToString[glfw.KeyF5] = "F5"
	keyToString[glfw.KeyF6] = "F6"
	keyToString[glfw.KeyF7] = "F7"
	keyToString[glfw.KeyF8] = "F8"
	keyToString[glfw.KeyF9] = "F9"
	keyToString[glfw.KeyF10] = "F10"
	keyToString[glfw.KeyF11] = "F11"
	keyToString[glfw.KeyF12] = "F12"
	keyToString[glfw.KeyPrintScreen] = "PRINTSCREEN"
	keyToString[glfw.KeyScrollLock] = "SCROLLLOCK"
	keyToString[glfw.KeyPause] = "PAUSE"
	keyToString[glfw.KeyInsert] = "INSERT"
	keyToString[glfw.KeyHome] = "HOME"
	keyToString[glfw.KeyPageUp] = "PAGEUP"
	keyToString[glfw.KeyDelete] = "DELETE"
	keyToString[glfw.KeyEnd] = "END"
	keyToString[glfw.KeyPageDown] = "PAGEDOWN"
	keyToString[glfw.KeyRight] = "RIGHT"
	keyToString[glfw.KeyLeft] = "LEFT"
	keyToString[glfw.KeyDown] = "DOWN"
	keyToString[glfw.KeyUp] = "UP"
	keyToString[glfw.KeyKPDivide] = "KP_DIVIDE"
	keyToString[glfw.KeyKPMultiply] = "KP_MULTIPLY"
	keyToString[glfw.KeyKPSubtract] = "KP_MINUS"
	keyToString[glfw.KeyKPAdd] = "KP_PLUS"
	keyToString[glfw.KeyKPEnter] = "KP_ENTER"
	keyToString[glfw.KeyKP1] = "KP_1"
	keyToString[glfw.KeyKP2] = "KP_2"
	keyToString[glfw.KeyKP3] = "KP_3"
	keyToString[glfw.KeyKP4] = "KP_4"
	keyToString[glfw.KeyKP5] = "KP_5"
	keyToString[glfw.KeyKP6] = "KP_6"
	keyToString[glfw.KeyKP7] = "KP_7"
	keyToString[glfw.KeyKP8] = "KP_8"
	keyToString[glfw.KeyKP9] = "KP_9"
	keyToString[glfw.KeyKP0] = "KP_0"
	keyToString[glfw.KeyKPDecimal] = "KP_PERIOD"
	keyToString[glfw.KeyKPEqual] = "KP_EQUALS"
	keyToString[glfw.KeyF13] = "F13"
	keyToString[glfw.KeyF14] = "F14"
	keyToString[glfw.KeyF15] = "F15"
	keyToString[glfw.KeyF16] = "F16"
	keyToString[glfw.KeyF17] = "F17"
	keyToString[glfw.KeyF18] = "F18"
	keyToString[glfw.KeyF19] = "F19"
	keyToString[glfw.KeyF20] = "F20"
	keyToString[glfw.KeyF21] = "F21"
	keyToString[glfw.KeyF22] = "F22"
	keyToString[glfw.KeyF23] = "F23"
	keyToString[glfw.KeyF24] = "F24"
	keyToString[glfw.KeyMenu] = "MENU"
	keyToString[glfw.KeyKPDecimal] = "KP_COMMA"
	keyToString[glfw.KeyLeftControl] = "LCTRL"
	keyToString[glfw.KeyLeftShift] = "LSHIFT"
	keyToString[glfw.KeyLeftAlt] = "LALT"
	keyToString[glfw.KeyLeftSuper] = "LGUI"
	keyToString[glfw.KeyRightControl] = "RCTRL"
	keyToString[glfw.KeyRightShift] = "RSHIFT"
	keyToString[glfw.KeyRightAlt] = "RALT"
	keyToString[glfw.KeyRightSuper] = "RGUI"
	lower := make(map[glfw.Key]string)
	for key, str := range keyToString {
		lower[key] = strings.ToLower(str)
	}
	keyToString = lower

	stringToKey = make(map[string]glfw.Key)
	for key, str := range keyToString {
		stringToKey[str] = key
	}
}
