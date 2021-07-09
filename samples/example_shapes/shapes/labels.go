package shapes

import "github.com/gonutz/prototype/draw"

type Label struct {
	*ScaledShape
	value     string
}

func NewLabel(name string) *Label {
	return &Label{newScaledShape(name), name}
}

func (l *Label) CarryEvent(window draw.Window) {
	if l.function != nil {
		l.function(window)
	}
}

func (l *Label) Paint(window draw.Window) {
	if HideNamesLocal.IsIn(l.name) {
		return
	}
	x, y := l.GetXY()
	window.FillRect(x, y, l.width, l.height, draw.White)
	window.DrawScaledText(l.value, x, y, l.sizeScale, draw.Red)
}

func (l *Label) SetValue(s string) {
	l.value = s
}

func (l *Label) SetSizeScale(s float32)  {
	l.sizeScale = s
}

func (t *TextField) GetValue() string {
	return t.value
}