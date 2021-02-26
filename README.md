UNDER CONSTRUCTION
==================

This repo is currently moving to use Go modules, this might take another day or two.

prototype
=========

Simply prototype 2D games using an easy, minimal interface that lets you draw simple primitives and images on the screen, easily handle mouse and keyboard events and play sounds.

![Games](https://github.com/gonutz/prototype/blob/master/samples/screenshots/games.png)

Installation
------------

Install the [Go programming language](https://golang.org/dl/). After clicking the download link you will be referred to the installation instructions for your specific operating system.

Install [Git](https://git-scm.com/downloads) and make it available in the PATH so the go tool can use it.

For Linux and OS X you need a C compiler installed. On Windows this is not necessary.

On Linux there are two backends available, GLFW and SDL2. For GLFW you need these libraries installed (tested on Linux Mint, other distros might be slightly different):

`libx11-dev libxrandr-dev libgl1-mesa-dev libxcursor-dev libxinerama-dev libxi-dev`

GLFW is used by default. To use SDL2 you need to add `-tags sdl2` to your Go builds, e.g. `go run -tags sdl2 main.go`. Also you need the SDL2 libraries installed:

`libsdl2-dev libsdl2-mixer-dev libsdl2-image-dev`


Install the library and samples by running the following on your command line:

	go get github.com/gonutz/prototype/...

Documentation
-------------

For a description of all library functions, see [the godoc page](http://godoc.org/github.com/gonutz/prototype/draw) for this project. Note that most of the functionality is in the Window interface and hence the descriptions are listed as code comments in the source for that type.

Example
-------

```Go
package main

import (
	"math"

	"github.com/gonutz/prototype/draw"
)

func main() {
	draw.RunWindow("Title", 640, 480, update)
}

func update(window draw.Window) {
	// find the screen center
	w, h := window.Size()
	centerX, centerY := w/2, h/2

	// draw a button in the center of the screen
	mouseX, mouseY := window.MousePosition()
	mouseInCircle := math.Hypot(float64(mouseX-centerX), float64(mouseY-centerY)) < 20
	color := draw.DarkRed
	if mouseInCircle {
		color = draw.Red
	}
	window.FillEllipse(centerX-20, centerY-20, 40, 40, color)
	window.DrawEllipse(centerX-20, centerY-20, 40, 40, draw.White)
	if mouseInCircle {
		window.DrawScaledText("Close!", centerX-40, centerY+25, 1.6, draw.Green)
	}

	// check all mouse clicks that happened during this frame
	for _, click := range window.Clicks() {
		dx, dy := click.X-centerX, click.Y-centerY
		squareDist := dx*dx + dy*dy
		if squareDist <= 20*20 {
			// close the window and end the application
			window.Close()
		}
	}
}
```
	
This example displays a window with a round button in the middle to close it. It demonstrates some basic drawing and event handling code.
