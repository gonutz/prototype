package draw

// UpdateFunction is used as a callback when creating a window. It is called
// regularly and you can do all your event handling and drawing in it.
type UpdateFunction func(window Window)

type Window interface {
	Close()

	WasKeyPressed(key string) bool
	IsKeyDown(key string) bool

	IsMouseDown(button MouseButton) bool
	Clicks() []MouseClick

	DrawPoint(x, y int, color Color)
	DrawLine(fromX, fromY, toX, toY int, color Color)
	DrawRect(x, y, width, height int, color Color)
	FillRect(x, y, width, height int, color Color)
	DrawEllipse(x, y, width, height int, color Color)
	FillEllipse(x, y, width, height int, color Color)
	DrawImageFile(path string, x, y int) error
	DrawImageFileTo(path string, x, y, w, h, degrees int) error
	DrawImageFilePortion(path string, srcX, srcY, srcW, srcH, toX, toY int) error

	GetTextSize(text string) (w, h int)
	GetScaledTextSize(text string, scale float32) (w, h int)
	DrawText(text string, x, y int, color Color)
	DrawScaledText(text string, x, y int, scale float32, color Color)

	PlaySoundFile(path string) error
}

type MouseClick struct {
	X, Y   int
	Button MouseButton
}

type MouseButton int

const (
	LeftButton MouseButton = iota
	MiddleButton
	RightButton
)

type Color struct{ R, G, B, A float32 }

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
