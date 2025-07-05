# drawsm

This tool helps manage WASM builds of your games.

Install it with `go install github.com/gonutz/prototype/cmd/drawsm@latest`.

To run your game locally as a WASM build, call `drawsm run` from your project
directory. This will build the game and serve it locally on port 8080. It will
open the default browser at the local URL.

To build your game use `drawsm build`. This will generate a template
`index.html` and copy your Go installation's `wasm_exec.js` to the project
directory. These files, along with the compilation output `main.wasm` are
necessary to server your game in the browser.

To serve the built files use `drawsm serve`. This serves the game locally on
port 8080 and opens the default browser at this location.

Call `drawsm help` for the tool's help text.
