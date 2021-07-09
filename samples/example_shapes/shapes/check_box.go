package shapes

import "github.com/gonutz/prototype/draw"

type CheckBox struct {
	*Basic
}

func (c *CheckBox) CarryEvent(window draw.Window) {
	if HideNamesLocal.IsIn(c.name) {
		return
	}
	click := window.Clicks()
	for _, mouseClick := range click {
		if mouseClick.Button==draw.LeftButton == false {
			continue
		}
		if c.IsIn(mouseClick.X, mouseClick.Y) {
			c.GetFunc()(window)
		}
	}
}

func (c *CheckBox) Paint(window draw.Window) {
	if HideNamesLocal.IsIn(c.name) {
		return
	}
	color := draw.LightRed
	if c.active {
		color = draw.LightGreen
	}
	window.FillRect(c.x, c.y, c.width, c.height, draw.White)
	window.FillEllipse(c.x, c.y, c.width, c.height, color)
}

func NewCheckBox(s string) *CheckBox {
	return &CheckBox{	Basic: newBasic(s)	}
}
