package main

import (
	"embed"
	"fmt"
	"io"
	"strings"

	"github.com/gonutz/prototype/draw"
)

//go:embed rsc/*
var rsc embed.FS

// Toggle loadFromEmbed to load from disk/URL (false) or from the embedded file
// system (true)
const loadFromEmbed = true

func main() {
	if loadFromEmbed {
		draw.OpenFile = func(path string) (io.ReadCloser, error) {
			return rsc.Open(path)
		}
	}

	var (
		fullscreen    bool
		blurImages    bool
		characters    string
		lastKeys      []draw.Key
		lastClick     draw.MouseClick
		wheelX        float64
		wheelY        float64
		cursorVisible = true
		textScale     = float32(1.0)
	)

	draw.RunWindow("Everything", 800, 600, func(window draw.Window) {

		if window.IsKeyDown(draw.KeyLeftControl) && window.WasKeyPressed(draw.KeyC) {
			window.Close()
		}

		if window.WasKeyPressed(draw.KeyI) {
			blurImages = !blurImages
		}
		window.BlurImages(blurImages)

		characters += window.Characters()

		n := len(window.Clicks())
		if n > 0 {
			lastClick = window.Clicks()[n-1]
		}

		for key := draw.KeyA; key <= draw.KeyPause; key++ {
			if window.WasKeyPressed(key) {
				lastKeys = append(lastKeys, key)
				if len(lastKeys) > 3 {
					lastKeys = lastKeys[len(lastKeys)-3:]
				}
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

		textScale += float32(window.MouseWheelY() / 10)
		if textScale < 0.1 {
			textScale = 0.1
		}

		if window.WasKeyPressed(draw.KeyF) {
			fullscreen = !fullscreen
			window.SetFullscreen(fullscreen)
		}

		if window.WasKeyPressed(draw.KeyC) {
			cursorVisible = !cursorVisible
			window.ShowCursor(cursorVisible)
		}

		if window.WasKeyPressed(draw.KeyS) {
			window.PlaySoundFile("rsc/sound.wav")
		}

		if window.WasKeyPressed(draw.KeyM) {
			window.PlaySoundFile("rsc/music.ogg")
		}

		mx, my := window.MousePosition()
		window.DrawLine(5, 400, mx, my, draw.White)
		window.DrawPoint(9, 390, draw.Yellow)
		window.FillRect(10, 440, 30, 40, draw.Red)
		window.DrawRect(50, 440, 30, 40, draw.Yellow)
		window.FillEllipse(90, 440, 30, 40, draw.Purple)
		window.DrawEllipse(130, 440, 30, 40, draw.Blue)

		window.DrawRect(170, 430, 0, 0, draw.Red)
		window.DrawRect(170, 434, 1, 0, draw.Red)
		window.DrawRect(170, 438, 0, 1, draw.Red)
		window.DrawRect(170, 442, 1, 1, draw.Red)
		window.DrawRect(170, 446, 2, 1, draw.LightRed)
		window.DrawRect(170, 450, 1, 2, draw.Red)
		window.DrawRect(170, 454, 2, 2, draw.LightRed)
		window.DrawRect(170, 458, 3, 1, draw.Red)
		window.DrawRect(170, 462, 1, 3, draw.LightRed)
		window.DrawRect(170, 466, 3, 2, draw.Red)
		window.DrawRect(170, 470, 2, 3, draw.LightRed)
		window.DrawRect(170, 474, 4, 3, draw.Red)
		window.DrawRect(170, 478, 3, 4, draw.LightRed)

		window.DrawEllipse(180, 430, 0, 0, draw.Blue)
		window.DrawEllipse(180, 434, 1, 0, draw.Blue)
		window.DrawEllipse(180, 438, 0, 1, draw.Blue)
		window.DrawEllipse(180, 442, 1, 1, draw.Blue)
		window.DrawEllipse(180, 446, 2, 1, draw.LightBlue)
		window.DrawEllipse(180, 450, 1, 2, draw.Blue)
		window.DrawEllipse(180, 454, 2, 2, draw.LightBlue)
		window.DrawEllipse(180, 458, 3, 1, draw.Blue)
		window.DrawEllipse(180, 462, 1, 3, draw.LightBlue)
		window.DrawEllipse(180, 466, 3, 2, draw.Blue)
		window.DrawEllipse(180, 470, 2, 3, draw.LightBlue)
		window.DrawEllipse(180, 474, 4, 3, draw.Blue)
		window.DrawEllipse(180, 478, 3, 4, draw.LightBlue)

		window.FillRect(190, 430, 0, 0, draw.Green)
		window.FillRect(190, 434, 1, 0, draw.Green)
		window.FillRect(190, 438, 0, 1, draw.Green)
		window.FillRect(190, 442, 1, 1, draw.Green)
		window.FillRect(190, 446, 2, 1, draw.LightGreen)
		window.FillRect(190, 450, 1, 2, draw.Green)
		window.FillRect(190, 454, 2, 2, draw.LightGreen)
		window.FillRect(190, 458, 3, 1, draw.Green)
		window.FillRect(190, 462, 1, 3, draw.LightGreen)
		window.FillRect(190, 466, 3, 2, draw.Green)
		window.FillRect(190, 470, 2, 3, draw.LightGreen)
		window.FillRect(190, 474, 4, 3, draw.Green)
		window.FillRect(190, 478, 3, 4, draw.LightGreen)

		window.FillEllipse(200, 430, 0, 0, draw.Yellow)
		window.FillEllipse(200, 434, 1, 0, draw.Yellow)
		window.FillEllipse(200, 438, 0, 1, draw.Yellow)
		window.FillEllipse(200, 442, 1, 1, draw.Yellow)
		window.FillEllipse(200, 446, 2, 1, draw.LightYellow)
		window.FillEllipse(200, 450, 1, 2, draw.Yellow)
		window.FillEllipse(200, 454, 2, 2, draw.LightYellow)
		window.FillEllipse(200, 458, 3, 1, draw.Yellow)
		window.FillEllipse(200, 462, 1, 3, draw.LightYellow)
		window.FillEllipse(200, 466, 3, 2, draw.Yellow)
		window.FillEllipse(200, 470, 2, 3, draw.LightYellow)
		window.FillEllipse(200, 474, 4, 3, draw.Yellow)
		window.FillEllipse(200, 478, 3, 4, draw.LightYellow)

		window.DrawLine(210, 440, 280, 480, draw.LightBrown)
		window.DrawLine(210, 445, 212, 447, draw.LightBlue)
		window.DrawLine(210, 450, 211, 451, draw.LightGreen)

		imgW, imgH, _ := window.ImageSize("rsc/meds.png")
		window.FillRect(9, 519, imgW+2, imgH+2, draw.DarkYellow)
		window.DrawImageFile("rsc/meds.png", 10, 520)
		window.DrawImageFilePart("rsc/meds.png", 32, 0, 16, 15, 100, 520, 3*16, 3*15, 45)
		window.DrawImageFileRotated("rsc/meds.png", 200, 520, -20)
		window.DrawImageFileTo("rsc/meds.png", 300, 520, 128, 64, 5)

		windowW, windowH := window.Size()

		text := "Ctrl+C: Close\n"
		text += fmt.Sprintf("Window Size: %d x %d\n", windowW, windowH)
		text += "C: Show/Hide Cursor (" + boolToString(cursorVisible) + ")\n"
		text += "F: Fullscreen (" + boolToString(fullscreen) + ")\n"
		text += "I: Blur Images (" + boolToString(blurImages) + ")\n"
		text += "S: Play Sound\n"
		text += "M: Play Music\n"
		text += "Text written so far: " + characters + "\n"

		lastKeyTexts := make([]string, len(lastKeys))
		for i, k := range lastKeys {
			lastKeyTexts[i] = k.String()
		}
		text += "Last typed keys: " + strings.Join(lastKeyTexts, " ") + "\n"

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
		textW, textH := window.GetScaledTextSize(text, textScale)
		window.FillRect(5, 5, textW, textH, draw.DarkPurple)
		window.DrawScaledText(text, 5, 5, textScale, draw.White)

		window.FillRect(500, 500, 500, 500, draw.Green)
	})
}

func boolToString(b bool) string {
	if b {
		return "on"
	}
	return "off"
}
