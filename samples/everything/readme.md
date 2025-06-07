This sample uses every API in the `prototype/draw` library.

It can be used to check that the library works the same on all platforms.

This sample was initially created for the WASM port and at the time of this
writing we include a copy of Go v1.24.2's `wasm_exec.js` which is needed to run
the code in a browser.
We walso have a `serve.go` which serves the sample to localhost:8080.
The `run` script will first start the desktop version of the sample, and once
that is closed, runs the server. This way we can compare the desktop version to
the browser version during WASM port development.
