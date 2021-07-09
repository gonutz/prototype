package main

import (
	sh"example_shapes/shapes"
	"github.com/gonutz/prototype/draw"
	"strconv"
)
func main() {
	all := initAll()
	_ = draw.RunWindow("shape example",500,500, func(window draw.Window) {
		for i := 0; i < len(all); i++ {
			all[i].Paint(window)
			all[i].CarryEvent(window)
		}
	})
}

func initAll() []sh.Shape {
	result := sh.MakeEmptyShapesArrayWithCapResetId(10)
	s,h := "show", "hide"
	b1 := sh.NewButtonRedWhiteGreen(s)
	b2 := sh.NewButtonRedWhiteGreen(h)
	b2.SetFunc(hide(h))
	b1.SetFunc(show(h))
	l1 := sh.NewLabel("name1")
	l1.SetFunc(func(window draw.Window) {
		l1.SetValue(strconv.Itoa(count))
	})
	b1.SetXY(200,10)
	b1.SetSize(150,50)
	b2.SetXY(200,60)
	b2.SetSize(150,50)
	l1.SetXY(200,110)
	result = append(append(append(result, b1), b2), l1)
	return result
}

var count = 1
func hide(name string)func(window draw.Window) {
	return func(window draw.Window) {
		if sh.HideNamesLocal.IsIn(name) {
			return
		}
		sh.HideNamesLocal.Add(name)
		count--
		//can anything else
	}
}

func show(name string) func(window draw.Window) {
	return func(window draw.Window) {
		if !sh.HideNamesLocal.IsIn(name) {
			return
		}
		sh.HideNamesLocal.Delete(name)
		count++
	}
}
