package shapes

import "github.com/gonutz/prototype/draw"

var (
	HideNamesLocal = HideNames(make(map[string]bool))
)

type Basic struct {
	x, y, endX, endY, width, height int
	id                              int64
	name                            string
	pressed, active                 bool
	function                        func(draw.Window)
}

type Shape interface {
	GetXY() (int,int)
	SetXY(x, y int)

	GetSize() (int,int)
	SetSize(width, height int)

	GetIdName() (int64, string)

	WasPressed() bool
	WasActive() bool
	IsIn(x int, y int) bool

	GetFunc() func(window draw.Window)
	SetFunc(function func(window draw.Window))

	Paint(window draw.Window)
	CarryEvent(window draw.Window)
}

var (
	id int64 = 0
)

func resetId() {
	id = 0
}

func MakeEmptyShapesArrayWithCapResetId(howMany int) []Shape {
	resetId()
	return make([]Shape, 0, howMany)
}
func (s *Basic) WasActive() bool {
	return s.active
}

func newBasic(name string) *Basic {
	id++
	return &Basic{name: name, id: id}
}
func (s *Basic) GetFunc() func(draw.Window) {
	return s.function
}
func (s *Basic) WasPressed() bool {
	return s.pressed
}

func (s *Basic) GetXY() (int,int) {
	return s.x, s.y
}

func (s *Basic) SetXY(x, y int) {
	s.x = x
	s.y = y
}

func (s *Basic) GetSize() (int,int) {
	return s.width, s.height
}

// SetSize
func (s *Basic) SetSize(width, height int) {
	s.width = width
	s.height = height
	s.endX = s.x + width
	s.endY = s.y + height
}

// GetIdName return id and Name
func (s *Basic) GetIdName() (int64, string) {
	return s.id, s.name
}

// SetPressed set element was clicked
func (s *Basic) SetPressed(b bool) {
	s.pressed = b
}

// SetActive set mouse is is
func (s *Basic) SetActive(b bool) {
	s.active = b
}

// SetFunc set own lamda func
func (s *Basic) SetFunc(function func(window draw.Window)) {
	s.function = function
}

// IsActive return whether mouse is in
func (s *Basic) IsActive() bool {
	return s.active
}

// IsIn return is position x,y in square, which start on self.x, self.y and end on  self.endX, self.endY
func (s *Basic) IsIn(x int, y int) bool {
	floatX, floatY := x, y
	return floatX >= s.x &&
		floatX <= s.endX &&
		floatY >= s.y &&
		floatY <= s.endY
}

// CarryEvent abstract method to care of event
func (s *Basic) CarryEvent(window draw.Window) {}

// Paint abstract method to paint
func (s *Basic) Paint(window draw.Window) {}
