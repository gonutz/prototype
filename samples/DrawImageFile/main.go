package main

import "github.com/gonutz/prototype/draw"

func main() {
	const (
		windowW, windowH = 800, 600
		tileW, tileH     = 101, 93
	)

	frameTime := 0
	err := draw.RunWindow("", windowW, windowH, func(window draw.Window) {
		frameTime++

		if window.WasKeyPressed(draw.KeyEscape) {
			window.Close()
		}

		window.FillRect(0, 0, windowW, windowH, draw.DarkGreen)

		{
			// DrawImageFile copies the whole image to the given position.
			window.DrawText("Sprite Sheet", 200, 0, draw.White)
			window.DrawImageFile("caveman.png", 0, 20)
		}

		{
			// DrawImageFilePart can be used to select a sub-rectangle and draw
			// it to a given position. Here we select the source rectangle based
			// on the current frame time to make a walk animation.
			window.DrawText("Push it!", 160, 255, draw.White)
			color := draw.DarkGreen
			color.G -= 10
			window.FillRect(0, 275, 500, tileH, color)
			window.DrawImageFilePart(
				"caveman.png",
				(2+(frameTime/10)%4)*tileW, 0*tileH, -tileW, tileH,
				-tileW+(2*frameTime)%500, 275, tileW, tileH,
				0,
			)
			window.DrawImageFilePart(
				"caveman.png",
				(2+(frameTime/10)%4)*tileW, 1*tileH, -tileW, tileH,
				-26-tileW+(2*frameTime)%500, 275, tileW, tileH,
				0,
			)
			window.FillRect(500-tileW, 275, 3*tileW, tileH, draw.DarkGreen)
		}

		{
			// DrawImageFilePart can scale the image.
			window.DrawText("Family", 130, windowH-tileH-30, draw.White)
			window.DrawImageFilePart(
				"caveman.png",
				1*tileW, 0*tileH, -tileW, tileH,
				10, windowH-tileH, tileW, tileH,
				0,
			)
			window.DrawImageFilePart(
				"caveman.png",
				0*tileW, 0*tileH, tileW, tileH,
				2*tileW, windowH-tileH*5/4, tileW*5/4, tileH*5/4,
				0,
			)
			window.DrawImageFilePart(
				"caveman.png",
				1*tileW, 0*tileH, -tileW, tileH,
				1*tileW, windowH-tileH/2, tileW/2, tileH/2,
				0,
			)
			window.DrawImageFilePart(
				"caveman.png",
				0*tileW, 0*tileH, tileW, tileH,
				2*tileW, windowH-tileH/3, tileW/3, tileH/3,
				0,
			)
		}

		{
			// DrawImageFilePart has a rotation in clockwise degrees. It rotates the
			// image about its center.
			window.DrawText(`
F
a
l
l
i
n
g

d
o
w
n
`, 720, 180, draw.White)
			window.DrawImageFilePart(
				"caveman.png",
				1*tileW, 1*tileH, -tileW, tileH,
				600, -100+(frameTime*4)%(windowH+200), tileW, tileH,
				frameTime,
			)
		}
	})

	if err != nil {
		panic(err)
	}
}
