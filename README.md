prototype
=========

Simply prototype 2D games using an easy, minimal interface that lets you draw simple primitives and images on the screen, easily handle mouse and keyboard events and play sounds.

Example
-------

	package main
	
	import "github.com/gonutz/prototype/draw"
	
	func main() {
		draw.RunWindow("Title", 640, 480, draw.Resizable, update)
	}
	
	func update(window draw.Window) {
		// find the screen center
		w, h := window.Size()
		centerX, centerY := w/2, h/2
	
		// draw a button in the center of the screen
		window.FillEllipse(centerX-20, centerY-20, 40, 40, draw.DarkRed)
		window.DrawEllipse(centerX-20, centerY-20, 40, 40, draw.White)
		window.DrawScaledText("Close!", centerX-40, centerY+25, 1.6, draw.Green)
	
		// check all mouse clicks that happened during this frams
		for _, click := range window.Clicks() {
			dx, dy := click.X-centerX, click.Y-centerY
			squareDist := dx*dx + dy*dy
			if squareDist <= 20*20 {
				// close the window and end the application
				window.Close()
			}
		}
	}

This displays a resizable window with a round button in the middle to close it. It shows some basic drawing and event handling code.