// +build !sdl2,!glfw

package draw

/*
#cgo CFLAGS: -DUNICODE -DWINVER=0x500
#include <Windows.h>

RAWKEYBOARD getRawKeyBoard(LPARAM lParam, int* valid) {
	*valid = 0;
	char buffer[sizeof(RAWINPUT)] = {};
	UINT size = sizeof(RAWINPUT);
	GetRawInputData((HRAWINPUT)lParam, RID_INPUT, buffer, &size, sizeof(RAWINPUTHEADER));

	// extract keyboard raw input data
	RAWINPUT* raw = (RAWINPUT*)buffer;
	if (raw->header.dwType == RIM_TYPEKEYBOARD)
	{
		*valid = 1;
		return raw->data.keyboard;
	}
}

void enableRawKeyboardInput(void* window) {
	RAWINPUTDEVICE inputDevice;
	inputDevice.usUsagePage = 0x01;
	inputDevice.usUsage = 0x06;
	inputDevice.dwFlags = 0;
	inputDevice.hwndTarget = (HWND)window;

	// NOTE Go 1.6 panics when doing this in Go because the window pointer will
	// be included in the C struct.
	RegisterRawInputDevices(&inputDevice, 1, sizeof(RAWINPUTDEVICE));
}
*/
import "C"

import (
	"bytes"
	"errors"
	"github.com/AllenDang/w32"
	"github.com/gonutz/d3d9"
	"github.com/gonutz/mixer"
	"github.com/gonutz/mixer/wav"
	"image"
	"image/draw"
	"image/png"
	"math"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

func init() {
	runtime.LockOSThread()
}

const (
	vertexFormat = d3d9.FVF_XYZRHW | d3d9.FVF_DIFFUSE | d3d9.FVF_TEX1
	vertexStride = 28

	fontTextureID = "///font"
)

var (
	windowOpenMutex      sync.Mutex
	windowIsOpen         bool
	globalWindow         *window
	fontCharW, fontCharH int
)

func RunWindow(title string, width, height int, update UpdateFunction) error {
	defer func() {
		windowOpenMutex.Lock()
		windowIsOpen = false
		windowOpenMutex.Unlock()
	}()

	var err error
	windowOpenMutex.Lock()
	if windowIsOpen {
		err = errors.New("a window is already open")
	}
	windowIsOpen = true
	windowOpenMutex.Unlock()
	if err != nil {
		return err
	}

	if err := mixer.Init(); err != nil {
		return err
	}
	defer mixer.Close()

	if err := d3d9.Init(); err != nil {
		return err
	}
	defer d3d9.Close()

	d3d, err := d3d9.Create(d3d9.SDK_VERSION)
	if err != nil {
		return err
	}
	defer d3d.Release()

	globalWindow = &window{
		running:  true,
		sounds:   make(map[string]mixer.SoundSource),
		textures: make(map[string]sizedTexture),
	}

	class := w32.WNDCLASSEX{
		Size:      C.sizeof_WNDCLASSEX,
		WndProc:   syscall.NewCallback(handleMessage),
		Cursor:    w32.LoadCursor(0, (*uint16)(unsafe.Pointer(uintptr(w32.IDC_ARROW)))),
		ClassName: syscall.StringToUTF16Ptr("GoPrototypeWindowClass"),
	}

	atom := w32.RegisterClassEx(&class)
	if atom == 0 {
		return errors.New("RegisterClassEx failed")
	}

	var style uint = w32.WS_OVERLAPPED | w32.WS_CAPTION | w32.WS_SYSMENU | w32.WS_VISIBLE
	var windowSize = w32.RECT{0, 0, int32(width), int32(height)}
	// NOTE MSDN says you cannot pass WS_OVERLAPPED to this function but it
	// seems to work (on XP and Windows 8.1 at least) in conjuntion with the
	// other flags
	if w32.AdjustWindowRect(&windowSize, style, false) {
		width = int(windowSize.Right - windowSize.Left)
		height = int(windowSize.Bottom - windowSize.Top)
	}

	var x, y int
	mode, err := d3d.GetAdapterDisplayMode(d3d9.ADAPTER_DEFAULT)
	if err == nil {
		x = int(mode.Width/2) - width/2
		y = int(mode.Height/2) - height/2
	}

	window := w32.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("GoPrototypeWindowClass"),
		nil,
		style,
		x, y, width, height,
		0, 0, 0, nil,
	)
	if window == 0 {
		return errors.New("CreateWindowEx failed")
	}
	globalWindow.handle = window
	w32.SetWindowText(window, title+" (D3D9)")
	w32.ShowWindow(w32.GetConsoleWindow(), w32.SW_HIDE)

	C.enableRawKeyboardInput(unsafe.Pointer(window))

	device, _, err := d3d.CreateDevice(
		d3d9.ADAPTER_DEFAULT,
		d3d9.DEVTYPE_HAL,
		unsafe.Pointer(window),
		d3d9.CREATE_SOFTWARE_VERTEXPROCESSING,
		d3d9.PRESENT_PARAMETERS{
			BackBufferFormat:     d3d9.FMT_UNKNOWN, // use current display format
			BackBufferCount:      1,
			Windowed:             true,
			SwapEffect:           d3d9.SWAPEFFECT_DISCARD,
			HDeviceWindow:        unsafe.Pointer(window),
			PresentationInterval: d3d9.PRESENT_INTERVAL_ONE, // enable VSyncx
		},
	)
	if err != nil {
		return err
	}
	defer device.Release()

	device.SetFVF(vertexFormat)
	device.SetRenderState(d3d9.RS_ZENABLE, d3d9.ZB_FALSE)
	device.SetRenderState(d3d9.RS_CULLMODE, d3d9.CULL_CCW)
	device.SetRenderState(d3d9.RS_LIGHTING, 0)
	device.SetRenderState(d3d9.RS_SRCBLEND, d3d9.BLEND_SRCALPHA)
	device.SetRenderState(d3d9.RS_DESTBLEND, d3d9.BLEND_INVSRCALPHA)
	device.SetRenderState(d3d9.RS_ALPHABLENDENABLE, 1)
	// use nearest neighbor texture filtering
	device.SetSamplerState(0, d3d9.SAMP_MINFILTER, d3d9.TEXF_NONE)
	device.SetSamplerState(0, d3d9.SAMP_MAGFILTER, d3d9.TEXF_NONE)

	device.SetTextureStageState(0, d3d9.TSS_COLOROP, d3d9.TOP_MODULATE)
	device.SetTextureStageState(0, d3d9.TSS_COLORARG1, d3d9.TA_TEXTURE)
	device.SetTextureStageState(0, d3d9.TSS_COLORARG2, d3d9.TA_DIFFUSE)

	device.SetTextureStageState(0, d3d9.TSS_ALPHAOP, d3d9.TOP_MODULATE)
	device.SetTextureStageState(0, d3d9.TSS_ALPHAARG1, d3d9.TA_TEXTURE)
	device.SetTextureStageState(0, d3d9.TSS_ALPHAARG2, d3d9.TA_DIFFUSE)

	device.SetTextureStageState(1, d3d9.TSS_COLOROP, d3d9.TOP_DISABLE)
	device.SetTextureStageState(1, d3d9.TSS_ALPHAOP, d3d9.TOP_DISABLE)

	globalWindow.device = device
	if err := globalWindow.loadFontTexture(); err != nil {
		return err
	}

	var msg w32.MSG
	w32.PeekMessage(&msg, 0, 0, 0, w32.PM_NOREMOVE)
	for msg.Message != w32.WM_QUIT && globalWindow.running {
		if w32.PeekMessage(&msg, 0, 0, 0, w32.PM_REMOVE) {
			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		} else {
			if err := device.Clear(nil, d3d9.CLEAR_TARGET, 0, 0, 0); err != nil {
				return err
			}
			if err := device.BeginScene(); err != nil {
				return err
			}

			update(globalWindow)
			if globalWindow.d3d9Error != nil {
				return globalWindow.d3d9Error
			}

			if err := device.EndScene(); err != nil {
				return err
			}
			if err := device.Present(nil, nil, nil, nil); err != nil {
				return err
			}

			globalWindow.finishFrame()
			// TODO check that VSync is active by measuring if one of the first
			// couple frames is much quicker than 1000/60 ms and if so => do a
			// sleep between frames
			//time.Sleep(1000 / 60 * time.Millisecond)
		}
	}

	for _, tex := range globalWindow.textures {
		tex.texture.Release()
	}

	globalWindow = nil
	return nil
}

type window struct {
	handle    w32.HWND
	device    d3d9.Device
	d3d9Error d3d9.Error
	running   bool
	mouse     struct{ x, y int }
	keyDown   [keyCount]bool
	mouseDown [mouseButtonCount]bool
	pressed   []Key
	clicks    []MouseClick
	sounds    map[string]mixer.SoundSource
	text      string
	textures  map[string]sizedTexture
}

func handleMessage(window w32.HWND, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case w32.WM_INPUT:
		var valid C.int
		kb := C.getRawKeyBoard(C.LPARAM(l), &valid)
		if valid == 0 {
			return 1
		}

		key, down := rawInputToKey(kb)
		if key != 0 {
			globalWindow.keyDown[key] = down
			if down {
				globalWindow.pressed = append(globalWindow.pressed, key)
				if key == KeyF4 && globalWindow.IsKeyDown(KeyLeftAlt) {
					globalWindow.Close()
				}
			}
		}
		return 1
	case w32.WM_CHAR:
		globalWindow.text += string(utf16.Decode([]uint16{uint16(w)})[0])
		return 1
	case w32.WM_MOUSEMOVE:
		globalWindow.mouse.x = int(int16(w32.LOWORD(uint32(l))))
		globalWindow.mouse.y = int(int16(w32.HIWORD(uint32(l))))
		return 1
	case w32.WM_LBUTTONDOWN:
		globalWindow.mouseEvent(LeftButton, true)
		return 1
	case w32.WM_LBUTTONUP:
		globalWindow.mouseEvent(LeftButton, false)
		return 1
	case w32.WM_RBUTTONDOWN:
		globalWindow.mouseEvent(RightButton, true)
		return 1
	case w32.WM_RBUTTONUP:
		globalWindow.mouseEvent(RightButton, false)
		return 1
	case w32.WM_MBUTTONDOWN:
		globalWindow.mouseEvent(MiddleButton, true)
		return 1
	case w32.WM_MBUTTONUP:
		globalWindow.mouseEvent(MiddleButton, false)
		return 1
	case w32.WM_DESTROY:
		w32.PostQuitMessage(0)
		return 1
	default:
		return w32.DefWindowProc(window, msg, w, l)
	}
}

func (w *window) Close() {
	w.running = false
}

func (w *window) Size() (int, int) {
	r := w32.GetClientRect(w.handle)
	return int(r.Right - r.Left), int(r.Bottom - r.Top)
}

func (w *window) WasKeyPressed(key Key) bool {
	for _, pressed := range w.pressed {
		if pressed == key {
			return true
		}
	}
	return false
}

func (w *window) IsKeyDown(key Key) bool {
	if key < 0 || key >= keyCount {
		return false
	}
	return w.keyDown[key]
}

func (w *window) Characters() string {
	return w.text
}

func (w *window) IsMouseDown(button MouseButton) bool {
	if button < 0 || button >= mouseButtonCount {
		return false
	}
	return w.mouseDown[button]
}

func (w *window) Clicks() []MouseClick {
	return w.clicks
}

func (w *window) MousePosition() (int, int) {
	return w.mouse.x, w.mouse.y
}

func (w *window) DrawPoint(x, y int, color Color) {
	data := [...]float32{
		float32(x), float32(y), 0, 1, colorToFloat32(color), 0, 0,
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_POINTLIST,
		1,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) DrawLine(fromX, fromY, toX, toY int, color Color) {
	if fromX == toX && fromY == toY {
		w.DrawPoint(fromX, fromY, color)
		return
	}

	col := colorToFloat32(color)
	fx, fy := float32(fromX), float32(fromY)
	fx2, fy2 := float32(toX), float32(toY)
	data := [...]float32{
		fx, fy, 0, 1, col, 0, 0,
		fx2, fy2, 0, 1, col, 0, 0,
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_LINELIST,
		1,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) DrawRect(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}

	w.FillRect(x, y, width, 1, color)
	w.FillRect(x, y, 1, height, color)
	w.FillRect(x+width-1, y, 1, height, color)
	w.FillRect(x, y+height-1, width, 1, color)
}

func (w *window) FillRect(x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}

	d3dColor := d3d9.ColorValue(color.R, color.G, color.B, color.A)
	var col float32 = *(*float32)(unsafe.Pointer(&d3dColor))
	fx, fy := float32(x), float32(y)
	fx2, fy2 := float32(x+width), float32(y+height)
	data := [...]float32{
		fx, fy, 0, 1, col, 0, 0,
		fx2, fy, 0, 1, col, 0, 0,
		fx, fy2, 0, 1, col, 0, 0,
		fx2, fy2, 0, 1, col, 0, 0,
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_TRIANGLESTRIP,
		2,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) DrawEllipse(x, y, width, height int, color Color) {
	w.ellipse(false, x, y, width, height, color)
}

func (w *window) FillEllipse(x, y, width, height int, color Color) {
	w.ellipse(true, x, y, width, height, color)
}

func (w *window) DrawImageFile(path string, x, y int) error {
	return w.renderImage(path, x, y, -1, -1, 0)
}

func (w *window) DrawImageFileRotated(path string, x, y, degrees int) error {
	return w.renderImage(path, x, y, -1, -1, degrees)
}

func (w *window) DrawImageFileTo(path string, x, y, width, height, degrees int) error {
	if width <= 0 || height <= 0 {
		return nil
	}

	return w.renderImage(path, x, y, width, height, degrees)
}

func (win *window) GetTextSize(text string) (w, h int) {
	return win.GetScaledTextSize(text, 1)
}

func (win *window) GetScaledTextSize(text string, scale float32) (w, h int) {
	if len(text) == 0 || scale <= 0 {
		return 0, 0
	}

	charW := int(float32(fontCharW)*scale + 0.5)
	charH := int(float32(fontCharH)*scale + 0.5)

	lines := strings.Split(text, "\n")
	maxLineW := 0
	for _, line := range lines {
		if len(line) > maxLineW {
			maxLineW = len(line)
		}
	}
	return charW * maxLineW, charH * len(lines)
}

func (w *window) DrawText(text string, x, y int, color Color) {
	w.DrawScaledText(text, x, y, 1, color)
}

func (w *window) DrawScaledText(text string, x, y int, scale float32, color Color) {
	if len(text) == 0 || scale <= 0 {
		return
	}

	texture := w.textures[fontTextureID]
	data := make([]float32, 0, vertexStride/4*len(text))
	width := int(float32(fontCharW)*scale + 0.5)
	height := int(float32(fontCharH)*scale + 0.5)
	col := colorToFloat32(color)
	destX, destY := x, y
	var charCount uint

	for _, char := range []byte(text) {
		if char == '\n' {
			destX = x
			destY += height
			continue
		}

		charCount++

		u := float32(char%16) / 16
		v := float32(char/16) / 16

		data = append(data,
			float32(destX)-0.5, float32(destY)-0.5, 0, 1, col, u, v,
			float32(destX+width)-0.5, float32(destY)-0.5, 0, 1, col, u+1.0/16, v,
			float32(destX)-0.5, float32(destY+height)-0.5, 0, 1, col, u, v+1.0/16,

			float32(destX)-0.5, float32(destY+height)-0.5, 0, 1, col, u, v+1.0/16,
			float32(destX+width)-0.5, float32(destY)-0.5, 0, 1, col, u+1.0/16, v,
			float32(destX+width)-0.5, float32(destY+height)-0.5, 0, 1, col, u+1.0/16, v+1.0/16,
		)

		destX += width
	}

	if err := w.device.SetTexture(0, texture.texture.BaseTexture); err != nil {
		w.d3d9Error = err
		return
	}

	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_TRIANGLELIST,
		charCount*2,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}

	// reset the texture
	if err := w.device.SetTexture(0, d3d9.BaseTexture{}); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) PlaySoundFile(path string) error {
	source, ok := w.sounds[path]
	if !ok {
		wave, err := wav.LoadFromFile(path)
		if err != nil {
			return err
		}

		source, err = mixer.NewSoundSource(wave)
		if err != nil {
			return err
		}

		w.sounds[path] = source
	}
	source.PlayOnce()
	return nil
}

func (w *window) mouseEvent(button MouseButton, down bool) {
	w.mouseDown[button] = down
	if down {
		w.clicks = append(w.clicks, MouseClick{
			X:      w.mouse.x,
			Y:      w.mouse.y,
			Button: button,
		})
	}
}

func getTextSizeInCharacters(text string) (int, int) {
	curCharsX, maxCharsX, lines := 0, 0, 1
	for _, c := range text {
		if c == '\n' {
			if curCharsX > maxCharsX {
				maxCharsX = curCharsX
			}
			lines++
			curCharsX = 0
		} else {
			curCharsX++
		}
	}
	return maxCharsX, lines
}

func (w *window) finishFrame() {
	w.pressed = w.pressed[0:0]
	w.clicks = w.clicks[0:0]
	w.text = ""
}

func colorToFloat32(color Color) float32 {
	d3dColor := d3d9.ColorValue(color.R, color.G, color.B, color.A)
	return *(*float32)(unsafe.Pointer(&d3dColor))
}

func (w *window) loadFontTexture() error {
	img, err := png.Decode(bytes.NewReader(bitmapFontWhitePng[:]))
	if err != nil {
		return err
	}

	fontCharW = img.Bounds().Dx() / 16
	fontCharH = img.Bounds().Dy() / 16

	return w.createTexture(fontTextureID, img)
}

func (w *window) loadTexture(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	img, err := png.Decode(file)
	if err != nil {
		return err
	}

	return w.createTexture(path, img)
}

func (w *window) createTexture(path string, img image.Image) error {
	var nrgba *image.NRGBA
	if i, ok := img.(*image.NRGBA); ok {
		nrgba = i
	} else {
		nrgba = image.NewNRGBA(img.Bounds())
		draw.Draw(nrgba, nrgba.Bounds(), img, image.ZP, draw.Src)
	}

	// swap r and b channel values
	for i := 0; i < len(nrgba.Pix); i += 4 {
		nrgba.Pix[i], nrgba.Pix[i+2] = nrgba.Pix[i+2], nrgba.Pix[i]
	}

	texture, err := w.device.CreateTexture(
		uint(nrgba.Bounds().Dx()),
		uint(nrgba.Bounds().Dy()),
		1,
		0,
		d3d9.FMT_A8R8G8B8,
		d3d9.POOL_MANAGED,
		nil,
	)
	if err != nil {
		return err
	}

	rect, err := texture.LockRect(0, nil, d3d9.LOCK_DISCARD)
	if err != nil {
		return err
	}
	rect.SetAllBytes(nrgba.Pix, nrgba.Stride)
	if err := texture.UnlockRect(0); err != nil {
		return err
	}

	w.textures[path] = sizedTexture{
		texture: texture,
		width:   nrgba.Bounds().Dx(),
		height:  nrgba.Bounds().Dy(),
	}

	return nil
}

type sizedTexture struct {
	texture       d3d9.Texture
	width, height int
}

func (w *window) renderImage(path string, x, y, width, height, degrees int) error {
	if _, ok := w.textures[path]; !ok {
		if err := w.loadTexture(path); err != nil {
			return err
		}
	}

	texture, ok := w.textures[path]
	if !ok {
		return errors.New("texture not found after loading: " + path)
	}

	if width == -1 && height == -1 {
		width, height = texture.width, texture.height
	}

	if err := w.device.SetTexture(0, texture.texture.BaseTexture); err != nil {
		return err
	}

	col := colorToFloat32(White)
	fx, fy, fw, fh := float32(x), float32(y), float32(width), float32(height)

	x1, y1 := -fw/2, -fh/2
	x2, y2 := fw/2, -fh/2
	x3, y3 := -fw/2, fh/2
	x4, y4 := fw/2, fh/2

	s, c := math.Sincos(float64(degrees) / 180 * math.Pi)
	sin, cos := float32(s), float32(c)
	x1, y1 = cos*x1-sin*y1, sin*x1+cos*y1
	x2, y2 = cos*x2-sin*y2, sin*x2+cos*y2
	x3, y3 = cos*x3-sin*y3, sin*x3+cos*y3
	x4, y4 = cos*x4-sin*y4, sin*x4+cos*y4

	dx := fx + fw/2 - 0.5
	dy := fy + fh/2 - 0.5
	data := [...]float32{
		x1 + dx, y1 + dy, 0, 1, col, 0, 0,
		x2 + dx, y2 + dy, 0, 1, col, 1, 0,
		x3 + dx, y3 + dy, 0, 1, col, 0, 1,
		x4 + dx, y4 + dy, 0, 1, col, 1, 1,
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_TRIANGLESTRIP,
		2,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}

	// reset the texture
	if err := w.device.SetTexture(0, d3d9.BaseTexture{}); err != nil {
		return err
	}

	return nil
}

func (w *window) ellipse(filled bool, x, y, width, height int, color Color) {
	if width <= 0 || height <= 0 {
		return
	}

	if width == 1 && height == 1 {
		w.DrawPoint(x, y, color)
		return
	}

	if width == 1 {
		w.DrawLine(x, y, x, y+height-1, color)
		return
	}
	if height == 1 {
		w.DrawLine(x, y, x+width-1, y, color)
		return
	}

	col := colorToFloat32(color)
	var data []float32

	if !filled {
		width--
		height--
	}
	a, b := float32(width)/2, float32(height)/2
	fx, fy := float32(x)+a, float32(y)+b

	var primitiveType d3d9.PRIMITIVETYPE
	var pointCount uint
	if filled {
		fx -= 0.5
		fy -= 0.5
		a -= 0.25
		b -= 0.25

		primitiveType = d3d9.PT_TRIANGLEFAN
		data = append(data, fx, fy, 0, 1, col, 0, 0)
		pointCount++
	} else {
		primitiveType = d3d9.PT_LINESTRIP
	}

	const stepCount = 50
	const dAngle = 2 * math.Pi / stepCount
	for i, angle := 0, 0.0; i <= stepCount; i, angle = i+1, angle+dAngle {
		sin, cos := math.Sincos(angle)
		x, y := a*float32(cos), b*float32(sin)
		data = append(data, fx+x, fy+y, 0, 1, col, 0, 0)
		pointCount++
	}

	var primitiveCount uint
	if filled {
		primitiveCount = pointCount - 2
	} else {
		primitiveCount = pointCount - 1
	}

	if err := w.device.DrawPrimitiveUP(
		primitiveType,
		primitiveCount,
		unsafe.Pointer(&data[0]),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func rawInputToKey(kb C.RAWKEYBOARD) (key Key, down bool) {
	virtualKey := C.USHORT(kb.VKey)
	scanCode := C.USHORT(kb.MakeCode)
	flags := kb.Flags

	down = flags&C.RI_KEY_BREAK == 0

	if virtualKey == 255 {
		// discard "fake keys" which are part of an escaped sequence
		return 0, down
	} else if virtualKey == C.VK_SHIFT {
		virtualKey = C.USHORT(C.MapVirtualKey(
			C.UINT(scanCode),
			C.MAPVK_VSC_TO_VK_EX,
		))
	} else if virtualKey == C.VK_NUMLOCK {
		// correct PAUSE/BREAK and NUM LOCK silliness, and set the extended
		// bit
		scanCode = C.USHORT(C.MapVirtualKey(
			C.UINT(virtualKey),
			C.MAPVK_VK_TO_VSC,
		) | 0x100)
	}

	isE0 := (flags & C.RI_KEY_E0) != 0
	isE1 := (flags & C.RI_KEY_E1) != 0

	if isE1 {
		// for escaped sequences, turn the virtual key into the correct scan code using MapVirtualKey.
		// however, MapVirtualKey is unable to map VK_PAUSE (this is a known bug), hence we map that by hand.
		if virtualKey == C.VK_PAUSE {
			scanCode = 0x45
		} else {
			scanCode = C.USHORT(C.MapVirtualKey(C.UINT(virtualKey), C.MAPVK_VK_TO_VSC))
		}
	}

	switch virtualKey {
	case C.VK_CONTROL:
		if isE0 {
			return KeyRightControl, down
		} else {
			return KeyLeftControl, down
		}
	case C.VK_MENU:
		if isE0 {
			return KeyRightAlt, down
		} else {
			return KeyLeftAlt, down
		}
	case C.VK_RETURN:
		if isE0 {
			return KeyNumEnter, down
		}
	case C.VK_INSERT:
		if !isE0 {
			return KeyNum0, down
		}
	case C.VK_HOME:
		if !isE0 {
			return KeyNum7, down
		}
	case C.VK_END:
		if !isE0 {
			return KeyNum1, down
		}
	case C.VK_PRIOR:
		if !isE0 {
			return KeyNum9, down
		}
	case C.VK_NEXT:
		if !isE0 {
			return KeyNum3, down
		}
	case C.VK_LEFT:
		if !isE0 {
			return KeyNum4, down
		}
	case C.VK_RIGHT:
		if !isE0 {
			return KeyNum6, down
		}
	case C.VK_UP:
		if !isE0 {
			return KeyNum8, down
		}
	case C.VK_DOWN:
		if !isE0 {
			return KeyNum2, down
		}
	case C.VK_CLEAR:
		if !isE0 {
			return KeyNum5, down
		}
	}

	if virtualKey >= 'A' && virtualKey <= 'Z' {
		return KeyA + Key(virtualKey-'A'), down
	} else if virtualKey >= '0' && virtualKey <= '9' {
		return Key0 + Key(virtualKey-'0'), down
	} else if virtualKey >= w32.VK_NUMPAD0 && virtualKey <= w32.VK_NUMPAD9 {
		return KeyNum0 + Key(virtualKey-w32.VK_NUMPAD0), down
	} else if virtualKey >= w32.VK_F1 && virtualKey <= w32.VK_F24 {
		return KeyF1 + Key(virtualKey-w32.VK_F1), down
	} else {
		switch virtualKey {
		case w32.VK_RETURN:
			return KeyEnter, down
		case w32.VK_LEFT:
			return KeyLeft, down
		case w32.VK_RIGHT:
			return KeyRight, down
		case w32.VK_UP:
			return KeyUp, down
		case w32.VK_DOWN:
			return KeyDown, down
		case w32.VK_ESCAPE:
			return KeyEscape, down
		case w32.VK_SPACE:
			return KeySpace, down
		case w32.VK_BACK:
			return KeyBackspace, down
		case w32.VK_TAB:
			return KeyTab, down
		case w32.VK_HOME:
			return KeyHome, down
		case w32.VK_END:
			return KeyEnd, down
		case w32.VK_NEXT:
			return KeyPageDown, down
		case w32.VK_PRIOR:
			return KeyPageUp, down
		case w32.VK_DELETE:
			return KeyDelete, down
		case w32.VK_INSERT:
			return KeyInsert, down
		case w32.VK_LSHIFT:
			return KeyLeftShift, down
		case w32.VK_RSHIFT:
			return KeyRightShift, down
		case w32.VK_PRINT:
			return KeyPrint, down
		case w32.VK_PAUSE:
			return KeyPause, down
		case w32.VK_CAPITAL:
			return KeyCapslock, down
		case w32.VK_MULTIPLY:
			return KeyNumMultiply, down
		case w32.VK_ADD:
			return KeyNumAdd, down
		case w32.VK_SUBTRACT:
			return KeyNumSubtract, down
		case w32.VK_DIVIDE:
			return KeyNumDivide, down
		}
	}

	return Key(0), false
}
