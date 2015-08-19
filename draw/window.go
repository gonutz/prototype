package draw

// UpdateFunction is used as a callback when creating a window. It is called
// at 60Hz and you do all your event handling and drawing in it.
type UpdateFunction func(window Window)

// Window provides functions to draw simple primitives and images, handle
// keyboard and mouse events and play sounds.
type Window interface {
	// Close closes the window which will stop the update loop after the current
	// frame (you will usually want to return from the update function after
	// calling Close or another frame will be displayed).
	Close()

	// Size returns the window's size in pixels.
	Size() (width, height int)

	// WasKeyPressed reports whether the specified key was pressed at any time
	// during the last frame. If the user presses a key and releases it in the
	// same frame, this function stores that information and will return true.
	//
	// The keys are addressed by a case-independent name. The following is a
	// description of possible values:
	//
	// "a" ... "z"       character keys are addressed by the respective character
	// "0" ... "9"       regular number keys
	// "kp_0" ... "kp_9" numbers on the num pad (or key pad, hence the prefix kp_)
	// "F1" ... "F24"    function keys
	// "enter"           the main enter/return key
	// "kp_enter"        the enter/return key on the num pad (key pad)
	// "lctrl"           left control key
	// "rctrl"           right control key
	// "lshift"          left shift key
	// "rshift"          right shift key
	// "lalt"            left alt key
	// "ralt"            right alt key
	// the following key names mean what they say
	// "left" "right" "up" "down"
	// "escape"
	// "space"
	// "backspace"
	// "tab"
	// "slash" "backslash"
	// "komma"
	// "minus"
	// "period" "kp_period"
	// "kp_plus" "kp_minus" "kp_divide" "kp_multiply"
	// "semicolon"
	// "leftbracket" "rightbracket"
	// "pageup" "pagedown"
	// "capslock"
	// "printscreen"
	// "scrolllock"
	// "pause"
	// "insert" "delete"
	// "home" "end"
	//
	// NOTE: it is not specified whether the key names actully correspond to the
	// current key layout. For example the German keyboard might report "z" for
	// the y key. To get an actual character, use WasCharTyped.
	WasKeyPressed(key string) bool

	// IsKeyDown reports whether the specified key is being held down at the
	// moment of calling this function. For a description of the key names, see
	// WasKeyPressed.
	IsKeyDown(key string) bool

	// WasCharTyped reports whether the specified rune was input during the last
	// frame. Producing this rune can be the result of multiply key strokes
	// (e.g. shfit+k for 'K').
	WasCharTyped(char rune) bool

	// IsMouseDown reports whether the specified button is down at the time of
	// the function call
	IsMouseDown(button MouseButton) bool

	// Clicks returns all MouseClicks that occurred during the last frame.
	Clicks() []MouseClick

	// MousePositoin returns the current mouse position in pixels at the time of
	// the function call. It is relative to the drawing area of the window.
	MousePosition() (x, y int)

	// DrawPoint draws a single point at the given screen position in pixels.
	DrawPoint(x, y int, color Color)

	// DrawLine draws a one pixel wide line from the first point to the second
	// (inclusive).
	DrawLine(fromX, fromY, toX, toY int, color Color)

	// DrawRect draws a one pixel wide rectangle outline.
	DrawRect(x, y, width, height int, color Color)

	// FillRect draws a filled rect.
	FillRect(x, y, width, height int, color Color)

	// DrawEllipse draws a one pixel wide ellipse.
	DrawEllipse(x, y, width, height int, color Color)

	// FillEllipse draws a filled ellipse.
	FillEllipse(x, y, width, height int, color Color)

	// DrawImageFile draws the untransformed image at the give position. If the
	// image file is not found an error is returned.
	DrawImageFile(path string, x, y int) error

	// DrawImageFileTo draws the image to the given screen rectangle, possibly
	// scaling it in either direction, and rotated it around the rectangles
	// center point by the given angle. The rotation is counterclockwise,
	DrawImageFileTo(path string, x, y, w, h, degrees int) error

	// GetTextSize returns the size the given text would have when begin drawn.
	GetTextSize(text string) (w, h int)

	// GetScaledTextSize returns the size the given text would have when begin
	// drawn at the given scale.
	GetScaledTextSize(text string, scale float32) (w, h int)

	// DrawText draws a text string. New line characters ('\n') are not drawn
	// but force a line break and the next character is drawn on the line below
	// starting again at x.
	DrawText(text string, x, y int, color Color)

	// DrawScaledText behaves as DrawText, but the text is scaled.
	DrawScaledText(text string, x, y int, scale float32, color Color)

	// PlaySoundFile only plays WAV sounds. If the file is not found an error is
	// returned.
	PlaySoundFile(path string) error
}

const (
	// Resizable means that the user can drag the borders of the window to resize it.
	Resizable = 1 << iota
)

// MouseClick is used to store mouse click events.
type MouseClick struct {
	// X and Y are the screen position in pixels, relative to the drawing area.
	X, Y   int
	Button MouseButton
}

type MouseButton int

const (
	LeftButton MouseButton = iota
	MiddleButton
	RightButton
)

// Color consists of four channels ranging form 0 to 1 each. A specifies the
// opacity, 1 being full opaque and 0 being fully transparent.
type Color struct{ R, G, B, A float32 }

// RGB creates a color with full opacity.
func RGB(r, g, b float32) Color {
	return Color{r, g, b, 1}
}

func RGBA(r, g, b, a float32) Color {
	return Color{r, g, b, a}
}

var Black = Color{0, 0, 0, 1}
var White = Color{1, 1, 1, 1}
var Gray = Color{0.5, 0.5, 0.5, 1}
var LightGray = Color{0.75, 0.75, 0.75, 1}
var DarkGray = Color{0.25, 0.25, 0.25, 1}
var Red = Color{1, 0, 0, 1}
var LightRed = Color{1, 0.5, 0.5, 1}
var DarkRed = Color{0.5, 0, 0, 1}
var Green = Color{0, 1, 0, 1}
var LightGreen = Color{0.5, 1, 0.5, 1}
var DarkGreen = Color{0, 0.5, 0, 1}
var Blue = Color{0, 0, 1, 1}
var LightBlue = Color{0.5, 0.5, 1, 1}
var DarkBlue = Color{0, 0, 0.5, 1}
var Purple = Color{1, 0, 1, 1}
var LightPurple = Color{1, 0.5, 1, 1}
var DarkPurple = Color{0.5, 0, 0.5, 1}
var Yellow = Color{1, 1, 0, 1}
var LightYellow = Color{1, 1, 0.5, 1}
var DarkYellow = Color{0.5, 0.5, 0, 1}
var Cyan = Color{0, 1, 1, 1}
var LightCyan = Color{0.5, 1, 1, 1}
var DarkCyan = Color{0, 0.5, 0.5, 1}
var Brown = Color{0.5, 0.2, 0, 1}
var LightBrown = Color{0.75, 0.3, 0, 1}
