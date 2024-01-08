package main

import (
	"fmt"
	"strings"

	"github.com/gonutz/prototype/draw"
)

func main() {
	var clicks [3 * 60]struct {
		text  string
		color draw.Color
	}
	clickColors := []draw.Color{
		draw.Purple,
		draw.Yellow,
		draw.Green,
		draw.Cyan,
	}
	nextClickColor := 0

	draw.RunWindow("Mouse", 800, 600, func(window draw.Window) {
		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}

		mouseX, mouseY := window.MousePosition()

		var buttons []string
		if window.IsMouseDown(draw.LeftButton) {
			buttons = append(buttons, "Left")
		}
		if window.IsMouseDown(draw.MiddleButton) {
			buttons = append(buttons, "Middle")
		}
		if window.IsMouseDown(draw.RightButton) {
			buttons = append(buttons, "Right")
		}

		var clickTexts []string
		for _, c := range window.Clicks() {
			which := ""
			if c.Button == draw.LeftButton {
				which = "Left"
			}
			if c.Button == draw.MiddleButton {
				which = "Middle"
			}
			if c.Button == draw.RightButton {
				which = "Right"
			}
			clickTexts = append(clickTexts, fmt.Sprintf(
				"%s Click at %d %d", which, c.X, c.Y,
			))
		}
		clickText := strings.Join(clickTexts, ", ")
		copy(clicks[1:], clicks[:])
		if clickText != "" {
			clicks[1].text = clickText
			clicks[1].color = clickColors[nextClickColor]
			nextClickColor = (nextClickColor + 1) % len(clickColors)
		}

		y := 10

		buttonsText := strings.Join(buttons, ", ")
		msg := fmt.Sprintf("%d / %d  %s", mouseX, mouseY, buttonsText)
		window.DrawText(msg, 10, y, draw.White)

		_, textH := window.GetTextSize("|")

		for i, c := range clicks {
			if c.text != "" {
				y += textH
				color := c.color
				color.A = float32(len(clicks)-1-i) / float32(len(clicks))
				window.DrawText(c.text, 10, y, color)
			}
		}
	})
}
