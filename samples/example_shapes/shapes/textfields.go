package shapes

import (
	"fmt"
	"github.com/gonutz/prototype/draw"
	"unicode"
	"unicode/utf8"
)

type TextField struct {
	*Label
	frame        int
	isErrorValue bool
}

func MakeEmptyTextFieldArrayWithCapResetId(howMany int) []*TextField {
	resetId()
	return make([]*TextField, 0, howMany)
}

func NewTextField(name string) *TextField {
	return &TextField{
		Label: NewLabel(name),
		frame:        0,
		isErrorValue: false,
	}
}

func (t *TextField) CarryEvent(window draw.Window) {
	if HideNamesLocal.IsIn(t.name) {
		return
	}
	t.carryClick(window)
	if t.active == false {
		return
	}
	t.backSpace(window)
	for _, r := range window.Characters() {
		t.execSelfFuncIfNotNil(window)
		if unicode.IsGraphic(r) {
			t.AddChar(r)
		}
	}
	t.careMouseEvent(window)
}

func (t *TextField) AddChar(char int32) {
	t.value = fmt.Sprint(t.value, string(char))
}

func (t *TextField) backSpace(window draw.Window) {
	if window.WasKeyPressed(draw.KeyBackspace) && t.active && t.value != "" {
		_, size := utf8.DecodeLastRuneInString(t.value)
		t.value = t.value[:len(t.value)-size]
	}
}

func (t *TextField) careMouseEvent(window draw.Window) {
	click := window.Clicks()
	length := len(click)
	if length == 0 {
		return
	}
	last := length - 1
	lastClick := click[last]
	if t.IsIn(lastClick.X, lastClick.Y) {
		t.active = true
		t.SetIsErrorValue(false)
	} else {
		t.active = false
	}
}

func (t *TextField) Paint(window draw.Window) {
	if HideNamesLocal.IsIn(t.name) {
		return
	}
	x, y := t.GetXY()
	color := draw.White
	if t.isErrorValue {
		color = draw.Red
	}
	window.FillRect(x, y, t.width, t.height, color)
	text := t.getTextCursor()
	window.DrawScaledText(text, t.x, t.y, 3, draw.Blue)
}

func (t *TextField) SetIsErrorValue(b bool) {
	t.isErrorValue = b
}

func (t *TextField) carryClick(window draw.Window) {
	x, y := window.MousePosition()
	if window.IsMouseDown(draw.LeftButton) {
		if t.IsIn(x, y) {
			t.SetActive(true)
		} else {
			t.SetActive(false)
		}
	}
}

func (t *TextField) getTextCursor() string {
	result := t.value
	if t.active {
		t.frame++
		if (t.frame)<30 {
			result = fmt.Sprint(result, "|")
		}else {
			if t.frame>60 {
				t.frame = 0
			}
		}
	}else {
		t.frame = 0
	}
	return result
}

func (t *TextField) execSelfFuncIfNotNil(window draw.Window) {
	if t.function != nil {
		t.function(window)
	}
}
