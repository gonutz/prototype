package shapes

import "github.com/gonutz/prototype/draw"

type Button struct {
	*ScaledShape
	aktiv, inaktiv, pressed *draw.Color
	interac bool
	frame int
}

func (b *Button) SetColours(aktiv, inaktiv, pressed *draw.Color) {
	b.aktiv = aktiv
	b.inaktiv = inaktiv
	b.pressed = pressed
}

func (b *Button) GetColour() draw.Color {
	if b.WasPressed() {
		return *b.pressed
	}
	if b.IsActive() {
		return *b.aktiv
	}
	return *b.inaktiv
}

func (b *Button) CarryEvent(window draw.Window) {
	if HideNamesLocal.IsIn(b.name) {
	}
	x, y := window.MousePosition()
	if b.IsIn(x, y) {
		b.SetActive(true)
		if window.IsMouseDown(draw.LeftButton) {
			b.SetPressed(true)
			b.executeIfFameWorkInterac(window)
		} else {
			b.executeIfWasRealaseDontInterac(window)
			b.SetPressed(false)
		}
	} else {
		b.SetPressed(false)
		b.SetActive(false)
	}
}

func (b *Button) executeIfFameWorkInterac(window draw.Window) *Button {
	if b.IsInterac()==false {
		return b
	}
	b.frame++
	if b.frame>=0 {
		b.frame=0
		b.GetFunc()(window)
	}
	return b
}

func (b *Button) Paint(window draw.Window){
	if HideNamesLocal.IsIn(b.name) {
		return
	}
	x, y := b.GetXY()
	width, height := b.GetSize()
	c := b.GetColour()
	window.FillRect(x, y, width, height, c)
	window.DrawScaledText(b.name, x-2, y-2, b.sizeScale, draw.LightBlue)
}

func (b *Button) IsInterac() bool {
	return b.interac
}

func (b *Button) SetInterac(bool bool){
	b.interac=bool
}

func (b *Button) executeIfWasRealaseDontInterac(window draw.Window){
	if b.IsInterac() {
		return
	}
	if b.WasPressed() {
		b.GetFunc()(window)
	}
}

func NewButton(name string) *Button {
	return &Button{ScaledShape:newScaledShape(name)}
}

func NewButtonRedWhiteGreen(name string) *Button {
	result := NewButton(name)
	result.SetColours(&draw.LightRed, &draw.White, &draw.LightGreen)
	return result
}
