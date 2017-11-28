prototype
=========

Simply prototype 2D games using an easy, minimal interface that lets you draw simple primitives and images on the screen, easily handle mouse and keyboard events and play sounds.

![Games](https://github.com/gonutz/prototype/blob/master/samples/screenshots/games.png)

Installation
------------

First you need to install the Go programming language from [this link](https://golang.org/dl/). After clicking the download link you will be referred to the installation instructions for your specific operating system. Follow these instructions so you have the GOROOT and GOPATH environment variables set in your system.

You of course also need git (for Windows, get it from [here](https://git-scm.com/downloads)).

If you use Linux or OS X you need a C compiler installed. On Windows, this is not necessary. If you are on Linux you probably already have GCC installed.

After installing all the tools you are ready to install this library and finally start prototyping. From the command line run:

	go get github.com/gonutz/prototype/...

This will install and build the library and also the sample games.

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