package main

import (
	"fmt"
	"strings"

	"github.com/gonutz/prototype/draw"
)

func main() {
	var (
		fullscreen    bool
		blurImages    bool
		blurText      bool
		characters    string
		lastKey       draw.Key
		lastClick     draw.MouseClick
		wheelX        float64
		wheelY        float64
		cursorVisible = true
	)

	draw.RunWindow("Everything", 800, 600, func(window draw.Window) {

		if window.IsKeyDown(draw.KeyLeftControl) && window.WasKeyPressed(draw.KeyC) {
			window.Close()
		}

		if window.WasKeyPressed(draw.KeyI) {
			blurImages = !blurImages
		}
		window.BlurImages(blurImages)

		if window.WasKeyPressed(draw.KeyT) {
			blurText = !blurText
		}
		window.BlurText(blurText)

		characters += window.Characters()

		n := len(window.Clicks())
		if n > 0 {
			lastClick = window.Clicks()[n-1]
		}

		for key := draw.KeyA; key <= draw.KeyPause; key++ {
			if window.WasKeyPressed(key) {
				lastKey = key
			}
		}

		keyDowns := ""
		for key := draw.KeyA; key <= draw.KeyPause; key++ {
			if window.IsKeyDown(key) {
				keyDowns += " " + key.String()
			}
		}

		mouseDowns := ""
		if window.IsMouseDown(draw.LeftButton) {
			mouseDowns += " Left"
		}
		if window.IsMouseDown(draw.MiddleButton) {
			mouseDowns += " Middle"
		}
		if window.IsMouseDown(draw.RightButton) {
			mouseDowns += " Right"
		}

		wheelX += window.MouseWheelX()
		wheelY += window.MouseWheelY()

		if window.WasKeyPressed(draw.KeyF) {
			fullscreen = !fullscreen
			window.SetFullscreen(fullscreen)
		}

		if window.WasKeyPressed(draw.KeyC) {
			cursorVisible = !cursorVisible
			fmt.Println("cursor", cursorVisible)
			window.ShowCursor(cursorVisible)
		}

		if window.WasKeyPressed(draw.KeyS) {
			window.PlaySoundFile("sound.wav")
		}

		mx, my := window.MousePosition()
		window.DrawLine(5, 400, mx, my, draw.White)
		window.DrawPoint(9, 390, draw.Yellow)
		window.FillRect(10, 440, 30, 40, draw.Red)
		window.DrawRect(50, 440, 30, 40, draw.Yellow)
		window.FillEllipse(90, 440, 30, 40, draw.Purple)
		window.DrawEllipse(130, 440, 30, 40, draw.Blue)
		imgW, imgH, _ := window.ImageSize("meds.png")
		window.FillRect(9, 519, imgW+2, imgH+2, draw.DarkYellow)
		window.DrawImageFile("meds.png", 10, 520)
		window.DrawImageFilePart("meds.png", 32, 0, 16, 15, 100, 520, 3*16, 3*15, 45)
		window.DrawImageFileRotated("meds.png", 200, 520, -20)
		window.DrawImageFileTo("meds.png", 300, 520, 128, 64, 5)

		windowW, windowH := window.Size()

		text := "Ctrl+C: Close\n"
		text += fmt.Sprintf("Window Size: %d x %d\n", windowW, windowH)
		text += "C: Show/Hide Cursor (" + boolToString(cursorVisible) + ")\n"
		text += "F: Fullscreen (" + boolToString(fullscreen) + ")\n"
		text += "I: Blur Images (" + boolToString(blurImages) + ")\n"
		text += "T: Blur Text (" + boolToString(blurText) + ")\n"
		text += "S: Play Sound\n"
		text += "Text written so far: " + characters + "\n"

		if lastKey != 0 {
			text += "Last typed key: " + lastKey.String() + "\n"
		}else {
			text += "Last typed key:\n"
		}

		text += "Pressed keys: " + keyDowns + "\n"
		text += "Pressed mouse buttons: " + mouseDowns + "\n"

		var defaultClick draw.MouseClick
		if lastClick != defaultClick {
			text += fmt.Sprintf(
				"Last Click: Button %v at %d,%d\n",
				lastClick.Button, lastClick.X, lastClick.Y,
			)
		} else {
			text += "Last Click:\n"
		}

		text += fmt.Sprintf("Mouse wheel: x %.2f, y %.2f\n", wheelX, wheelY)

		text = strings.TrimSuffix(text, "\n")
		textW, textH := window.GetScaledTextSize(text, 1.5)
		window.FillRect(5, 5, textW, textH, draw.DarkPurple)
		window.DrawScaledText(text, 5, 5, 1.5, draw.White)

	})
}

func boolToString(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
