# prototype

Simply prototype 2D games using an easy, minimal interface that lets you draw simple primitives and images on the screen, easily handle mouse and keyboard events, and play sounds.

![Games](https://github.com/gonutz/prototype/blob/master/samples/screenshots/games.png)

## Installation

Install the [Go programming language](https://golang.org/dl/). After clicking the download link you will be referred to the installation instructions for your specific operating system.

Install [Git](https://git-scm.com/downloads) and make it available in the PATH so the Go tool can use it.

For Linux and macOS, you need a C compiler installed. On Windows this is not necessary.

### Supported Targets

The prototype framework supports multiple targets:

#### Windows (default)

- Uses Direct3D 9
- No additional dependencies needed

#### Linux/macOS (GLFW backend)

- Uses OpenGL via GLFW
- Install required packages (example for Ubuntu/Debian):
  ```sh
  sudo apt install libx11-dev libxrandr-dev libgl1-mesa-dev libxcursor-dev libxinerama-dev libxi-dev
  ```

#### Linux (SDL2 backend)

- Install SDL2 libraries:
  ```sh
  sudo apt install libsdl2-dev libsdl2-mixer-dev libsdl2-image-dev
  ```
- Use build tag:
  ```sh
  go run -tags sdl2 main.go
  ```

#### WebAssembly (experimental)

- Implemented via HTML5 Canvas and Web Audio using `syscall/js`
- Requires Go 1.21+
- Compile using:
  ```sh
  GOOS=js GOARCH=wasm go build -o main.wasm
  ```
- Use a simple HTML wrapper with a canvas and `wasm_exec.js` (from Go installation):
  ```html
  <canvas id="gameCanvas" width="800" height="600" style="touch-action:none;"></canvas>
  <script src="wasm_exec.js"></script>
  <script>
    const go = new Go();
    WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
      go.run(result.instance);
    });
  </script>
  ```

> [!NOTE]
> The required `wasm_exec.js` file is included in the Go installation. You can find it in the `$(go env GOROOT)/misc/wasm` or `$(go env GOROOT)/lib/wasm` directory. Copy it to your project directory.

To serve locally:

```sh
python3 -m http.server
```

## Installation (Library & Samples)

Install the library and samples by running:

```sh
go get github.com/gonutz/prototype/...
```

## Documentation

For a description of all library functions, see [the GoDoc page](http://godoc.org/github.com/gonutz/prototype/draw). Most functionality is in the `Window` interface, and documented via code comments.

## Example

```go
package main

import (
	"math"

	"github.com/gonutz/prototype/draw"
)

func main() {
	draw.RunWindow("Title", 640, 480, update)
}

func update(window draw.Window) {
	w, h := window.Size()
	centerX, centerY := w/2, h/2

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

	for _, click := range window.Clicks() {
		dx, dy := click.X-centerX, click.Y-centerY
		if dx*dx+dy*dy <= 20*20 {
			window.Close()
		}
	}
}
```

This example displays a window with a round button in the middle to close it. It demonstrates basic drawing and event handling.
