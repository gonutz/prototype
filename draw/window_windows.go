// +build !sdl2,!glfw

package draw

import (
	"bytes"
	"errors"
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
	"unicode/utf8"
	"unsafe"

	"github.com/gonutz/d3d9"
	"github.com/gonutz/mixer"
	"github.com/gonutz/mixer/wav"
	"github.com/gonutz/w32"
)

func init() {
	runtime.LockOSThread()
}

const (
	vertexFormat = d3d9.FVF_XYZRHW | d3d9.FVF_DIFFUSE | d3d9.FVF_TEX1
	vertexStride = 28

	windowedStyle = w32.WS_OVERLAPPED | w32.WS_CAPTION | w32.WS_SYSMENU | w32.WS_VISIBLE

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

	soundOn := false
	if err := mixer.Init(); err == nil {
		soundOn = true
		defer mixer.Close()
	}

	d3d, err := d3d9.Create(d3d9.SDK_VERSION)
	if err != nil {
		return err
	}
	defer d3d.Release()

	globalWindow = &window{
		running:  true,
		soundOn:  soundOn,
		sounds:   make(map[string]mixer.SoundSource),
		textures: make(map[string]sizedTexture),
	}

	class := w32.WNDCLASSEX{
		WndProc:   syscall.NewCallback(handleMessage),
		Cursor:    w32.LoadCursor(0, (*uint16)(unsafe.Pointer(uintptr(w32.IDC_ARROW)))),
		ClassName: syscall.StringToUTF16Ptr("GoPrototypeWindowClass"),
	}

	atom := w32.RegisterClassEx(&class)
	if atom == 0 {
		return errors.New("RegisterClassEx failed")
	}
	defer w32.UnregisterClassAtom(atom, w32.GetModuleHandle(""))

	var windowSize = w32.RECT{0, 0, int32(width), int32(height)}
	// NOTE MSDN says you cannot pass WS_OVERLAPPED to this function but it
	// seems to work (on XP and Windows 8.1 at least) in conjuntion with the
	// other flags
	if w32.AdjustWindowRect(&windowSize, windowedStyle, false) {
		width = int(windowSize.Width())
		height = int(windowSize.Height())
	}

	// list all monitors and find the largest one so we can make our back buffer
	// handle any fullscreen size.
	monitorCount := d3d.GetAdapterCount()
	backBufferWidth, backBufferHeight := width, height
	for i := uint(0); i < monitorCount; i++ {
		if mode, err := d3d.GetAdapterDisplayMode(i); err == nil {
			w, h := int(mode.Width), int(mode.Height)
			if w > backBufferWidth {
				backBufferWidth = w
			}
			if h > backBufferHeight {
				backBufferHeight = h
			}
		}
	}

	// find the first monitor that is large enough to fit the window, if none is
	// found we just use the default monitor
	var selectedMonitor uint = d3d9.ADAPTER_DEFAULT
	for i := uint(0); i < monitorCount; i++ {
		mode, err := d3d.GetAdapterDisplayMode(i)
		if err == nil && int(mode.Width) >= width && int(mode.Height) >= height {
			selectedMonitor = i
			break
		}
	}
	// center the window in the monitor, if any of these functions fail, x,y
	// will simply be 0,0 which is fine in that case
	var x, y int
	refreshRate := 60 // default to 60 Hz in case we cannot query the monitor
	mode, err := d3d.GetAdapterDisplayMode(selectedMonitor)
	if err == nil {
		if mode.RefreshRate != 0 { // 0 is some invalid default value
			refreshRate = int(mode.RefreshRate)
		}
		monitor := d3d.GetAdapterMonitor(selectedMonitor)
		if monitor != 0 {
			var info w32.MONITORINFO
			if w32.GetMonitorInfo(w32.HMONITOR(monitor), &info) {
				workW := int(info.RcWork.Width())
				workH := int(info.RcWork.Height())
				x = int(info.RcWork.Left) + (workW-width)/2
				y = int(info.RcWork.Top) + (workH-height)/2
			}
		}
	}

	window := w32.CreateWindowEx(
		0,
		syscall.StringToUTF16Ptr("GoPrototypeWindowClass"),
		nil,
		windowedStyle,
		x, y, width, height,
		0, 0, 0, nil,
	)
	if window == 0 {
		return errors.New("CreateWindowEx failed")
	}
	defer w32.DestroyWindow(window)
	globalWindow.handle = window
	w32.SetWindowText(window, title)

	// hide the console window if double-clicking on the executable
	hideConsoleWindow()

	// enable raw keyboard input which allows us to handle keys like
	// shift/control/alt
	if !w32.RegisterRawInputDevices(w32.RAWINPUTDEVICE{
		UsagePage: 0x01,
		Usage:     0x06,
		Target:    window,
	}) {
		return errors.New("RegisterRawInputDevices failed")
	}

	device, presentParams, err := d3d.CreateDevice(
		d3d9.ADAPTER_DEFAULT,
		d3d9.DEVTYPE_HAL,
		d3d9.HWND(window),
		d3d9.CREATE_SOFTWARE_VERTEXPROCESSING,
		d3d9.PRESENT_PARAMETERS{
			BackBufferFormat:     d3d9.FMT_UNKNOWN, // use current display format
			BackBufferWidth:      uint32(backBufferWidth),
			BackBufferHeight:     uint32(backBufferHeight),
			BackBufferCount:      1,
			Windowed:             1,
			SwapEffect:           d3d9.SWAPEFFECT_COPY, // so Present can use rects
			HDeviceWindow:        d3d9.HWND(window),
			PresentationInterval: d3d9.PRESENT_INTERVAL_ONE, // enable VSync
		},
	)
	if err != nil {
		return err
	}
	defer device.Release()

	setRenderState := func() {
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
	}
	setRenderState()

	globalWindow.device = device
	if err := globalWindow.loadFontTexture(); err != nil {
		return err
	}

	// we want to update the game with 60 Hz, if the monitor has e.g. 120 Hz, we
	// need to update every other vsync, in case of 30 Hz we need to update
	// twice per vsync
	if 58 <= refreshRate && refreshRate <= 62 {
		// close enough, treat it like the 60 Hz that we want
		refreshRate = 60
	}
	updatesPerVsync := 60.0 / float32(refreshRate)
	nextUpdate := updatesPerVsync

	// TODO right now we just assume that the refresh setting the DX9 gives us
	// is correct but maybe the user changed some driver setting that we do not
	// know of; in this case the actual refresh rate might be different from
	// what D3D9 reports; we could measure some frames and estimate the actual
	// refresh rate, then compensate for it

	deviceIsLost := false

	var msg w32.MSG
	w32.PeekMessage(&msg, 0, 0, 0, w32.PM_NOREMOVE)
	for msg.Message != w32.WM_QUIT && globalWindow.running {
		if w32.PeekMessage(&msg, 0, 0, 0, w32.PM_REMOVE) {
			w32.TranslateMessage(&msg)
			w32.DispatchMessage(&msg)
		} else {
			if deviceIsLost {
				_, err = device.Reset(presentParams)
				if err == nil {
					deviceIsLost = false
					setRenderState()
				}
			}

			if !deviceIsLost {
				if err := device.BeginScene(); err != nil {
					return err
				}

				var wasUpdated bool
				for nextUpdate > 0 {
					// clear the screen to black before the update
					globalWindow.FillRect(0, 0, width, height, Black)
					update(globalWindow)
					wasUpdated = true
					nextUpdate -= 1
				}
				nextUpdate += updatesPerVsync

				if globalWindow.d3d9Error != nil {
					return globalWindow.d3d9Error
				}

				if err := device.EndScene(); err != nil {
					return err
				}
				windowW, windowH := globalWindow.Size()
				r := &d3d9.RECT{0, 0, int32(windowW), int32(windowH)}
				if presentErr := device.Present(r, r, 0, nil); presentErr != nil {
					if presentErr.Code() == d3d9.ERR_DEVICELOST {
						deviceIsLost = true
					} else {
						return presentErr
					}
				}

				if wasUpdated {
					globalWindow.finishFrame()
				}
			}
		}
	}

	for _, tex := range globalWindow.textures {
		tex.texture.Release()
	}

	globalWindow = nil
	return nil
}

func hideConsoleWindow() {
	console := w32.GetConsoleWindow()
	if console == 0 {
		return // no console attached
	}
	// If this application is the process that created the console window, then
	// this program was not compiled with the -H=windowsgui flag and on start-up
	// it created a console along with the main application window. In this case
	// hide the console window.
	// See
	// http://stackoverflow.com/questions/9009333/how-to-check-if-the-program-is-run-from-a-console
	// and thanks to
	// https://github.com/hajimehoshi
	// for the tip.
	_, consoleProcID := w32.GetWindowThreadProcessId(console)
	if w32.GetCurrentProcessId() == consoleProcID {
		w32.ShowWindowAsync(console, w32.SW_HIDE)
	}
}

type window struct {
	handle       w32.HWND
	device       *d3d9.Device
	d3d9Error    d3d9.Error
	running      bool
	isFullscreen bool
	windowed     w32.WINDOWPLACEMENT
	mouse        struct{ x, y int }
	wheelX       float64
	wheelY       float64
	keyDown      [keyCount]bool
	mouseDown    [mouseButtonCount]bool
	pressed      []Key
	clicks       []MouseClick
	soundOn      bool
	sounds       map[string]mixer.SoundSource
	text         string
	textures     map[string]sizedTexture
}

func handleMessage(window w32.HWND, msg uint32, w, l uintptr) uintptr {
	switch msg {
	case w32.WM_INPUT:
		raw, ok := w32.GetRawInputData(w32.HRAWINPUT(l), w32.RID_INPUT)
		if !ok {
			return 1
		}
		if raw.Header.Type != w32.RIM_TYPEKEYBOARD {
			return 1
		}
		key, down := rawInputToKey(raw.GetKeyboard())
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
	case w32.WM_MOUSEWHEEL:
		globalWindow.wheelY += float64(int16(w32.HIWORD(uint32(w)))) / 120.0
		return 1
	case w32.WM_MOUSEHWHEEL:
		globalWindow.wheelX += float64(int16(w32.HIWORD(uint32(w)))) / 120.0
		return 1
	case w32.WM_DESTROY:
		if globalWindow != nil {
			globalWindow.running = false
		}
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

func (w *window) SetFullscreen(f bool) {
	if f == w.isFullscreen {
		return
	}

	if f {
		w.windowed = enableFullscreen(w.handle)
	} else {
		disableFullscreen(w.handle, w.windowed)
	}

	w.isFullscreen = f
}

// enableFullscreen makes the window a borderless window that covers the full
// area of the monitor under the window.
// It returns the previous window placement. Store that value and use it with
// disableFullscreen to reset the window to what it was before.
func enableFullscreen(window w32.HWND) (windowed w32.WINDOWPLACEMENT) {
	style := w32.GetWindowLong(window, w32.GWL_STYLE)
	var monitorInfo w32.MONITORINFO
	monitor := w32.MonitorFromWindow(window, w32.MONITOR_DEFAULTTOPRIMARY)
	if w32.GetWindowPlacement(window, &windowed) &&
		w32.GetMonitorInfo(monitor, &monitorInfo) {
		w32.SetWindowLong(
			window,
			w32.GWL_STYLE,
			uint32(style & ^w32.WS_OVERLAPPEDWINDOW),
		)
		w32.SetWindowPos(
			window,
			0,
			int(monitorInfo.RcMonitor.Left),
			int(monitorInfo.RcMonitor.Top),
			int(monitorInfo.RcMonitor.Right-monitorInfo.RcMonitor.Left),
			int(monitorInfo.RcMonitor.Bottom-monitorInfo.RcMonitor.Top),
			w32.SWP_NOOWNERZORDER|w32.SWP_FRAMECHANGED,
		)
	}
	w32.ShowCursor(false)
	return
}

// disableFullscreen makes the window have a border, title and the close button
// and places it at the position given by the window placement parameter.
// Use this in conjunction with enableFullscreen to toggle a window's fullscreen
// state.
func disableFullscreen(window w32.HWND, placement w32.WINDOWPLACEMENT) {
	w32.SetWindowLong(window, w32.GWL_STYLE, windowedStyle)
	w32.SetWindowPlacement(window, &placement)
	w32.SetWindowPos(window, 0, 0, 0, 0, 0,
		w32.SWP_NOMOVE|w32.SWP_NOSIZE|w32.SWP_NOZORDER|
			w32.SWP_NOOWNERZORDER|w32.SWP_FRAMECHANGED,
	)
	w32.ShowCursor(true)
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

func (w *window) MouseWheelX() float64 {
	return w.wheelX
}

func (w *window) MouseWheelY() float64 {
	return w.wheelY
}

func (w *window) DrawPoint(x, y int, color Color) {
	data := [...]float32{
		float32(x), float32(y), 0, 1, colorToFloat32(color), 0, 0,
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_POINTLIST,
		1,
		uintptr(unsafe.Pointer(&data[0])),
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
		uintptr(unsafe.Pointer(&data[0])),
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
		uintptr(unsafe.Pointer(&data[0])),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) DrawEllipse(x, y, width, height int, color Color) {
	outline := ellipseOutline(x, y, width, height)
	if len(outline) == 0 {
		return
	}
	data := make([]float32, len(outline)*7)
	col := colorToFloat32(color)
	for i := range outline {
		j := i * 7
		data[j+0] = float32(outline[i].x)
		data[j+1] = float32(outline[i].y)
		data[j+2] = 0
		data[j+3] = 1
		data[j+4] = col
		data[j+5] = 0
		data[j+6] = 0
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_POINTLIST,
		uint(len(outline)),
		uintptr(unsafe.Pointer(&data[0])),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) FillEllipse(x, y, width, height int, color Color) {
	area := ellipseArea(x, y, width, height)
	if len(area) == 0 {
		return
	}
	col := colorToFloat32(color)
	data := make([]float32, len(area)*7)
	for i := range area {
		j := i * 7
		data[j+0] = float32(area[i].x)
		data[j+1] = float32(area[i].y)
		data[j+2] = 0
		data[j+3] = 1
		data[j+4] = col
		data[j+5] = 0
		data[j+6] = 0
	}
	// now offset every right point in each line by +0.5, otherwise they might
	// not be fully visible
	for i := 8; i < len(data); i += 14 {
		data[i-1] += 0.5
	}
	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_LINELIST,
		uint(len(area)/2),
		uintptr(unsafe.Pointer(&data[0])),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}
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
	charW := int(float32(fontCharW)*scale + 0.5)
	charH := int(float32(fontCharH)*scale + 0.5)

	lines := strings.Split(text, "\n")
	maxLineW := 0
	for _, line := range lines {
		w := utf8.RuneCountInString(line)
		if w > maxLineW {
			maxLineW = w
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

	for _, r := range text {
		if r == '\n' {
			destX = x
			destY += height
			continue
		}
		r = runeToFont(r)

		charCount++

		u := float32(r%16) / 16
		v := float32(r/16) / 16

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

	if err := w.device.SetTexture(0, texture.texture); err != nil {
		w.d3d9Error = err
		return
	}

	if err := w.device.DrawPrimitiveUP(
		d3d9.PT_TRIANGLELIST,
		charCount*2,
		uintptr(unsafe.Pointer(&data[0])),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}

	// reset the texture
	if err := w.device.SetTexture(0, nil); err != nil {
		w.d3d9Error = err
	}
}

func (w *window) PlaySoundFile(path string) error {
	if !w.soundOn {
		return errors.New("sound mixer could not be initialized")
	}
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
	w.wheelX = 0
	w.wheelY = 0
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
		0,
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
	texture       *d3d9.Texture
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

	if err := w.device.SetTexture(0, texture.texture); err != nil {
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
		uintptr(unsafe.Pointer(&data[0])),
		vertexStride,
	); err != nil {
		w.d3d9Error = err
	}

	// reset the texture
	if err := w.device.SetTexture(0, nil); err != nil {
		return err
	}

	return nil
}

func rawInputToKey(kb w32.RAWKEYBOARD) (key Key, down bool) {
	virtualKey := kb.VKey
	scanCode := kb.MakeCode
	flags := kb.Flags

	down = flags&w32.RI_KEY_BREAK == 0

	if virtualKey == 255 {
		// discard "fake keys" which are part of an escaped sequence
		return 0, down
	} else if virtualKey == w32.VK_SHIFT {
		virtualKey = uint16(w32.MapVirtualKey(
			uint(scanCode),
			w32.MAPVK_VSC_TO_VK_EX,
		))
	}

	isE0 := (flags & w32.RI_KEY_E0) != 0

	switch virtualKey {
	case w32.VK_CONTROL:
		if isE0 {
			return KeyRightControl, down
		} else {
			return KeyLeftControl, down
		}
	case w32.VK_MENU:
		if isE0 {
			return KeyRightAlt, down
		} else {
			return KeyLeftAlt, down
		}
	case w32.VK_RETURN:
		if isE0 {
			return KeyNumEnter, down
		}
	case w32.VK_INSERT:
		if !isE0 {
			return KeyNum0, down
		}
	case w32.VK_HOME:
		if !isE0 {
			return KeyNum7, down
		}
	case w32.VK_END:
		if !isE0 {
			return KeyNum1, down
		}
	case w32.VK_PRIOR:
		if !isE0 {
			return KeyNum9, down
		}
	case w32.VK_NEXT:
		if !isE0 {
			return KeyNum3, down
		}
	case w32.VK_LEFT:
		if !isE0 {
			return KeyNum4, down
		}
	case w32.VK_RIGHT:
		if !isE0 {
			return KeyNum6, down
		}
	case w32.VK_UP:
		if !isE0 {
			return KeyNum8, down
		}
	case w32.VK_DOWN:
		if !isE0 {
			return KeyNum2, down
		}
	case w32.VK_CLEAR:
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
