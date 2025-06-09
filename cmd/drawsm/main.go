package main

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

func help() {
	fmt.Println(`drawsm is a tool for managing github.com/prototype/draw WASM builds.

Usage:

  drawsm <command>

The commands are:

  build

    Generates index.html, wasm_exec.js and main.wasm in the current
    project. Serve these files to generate the game website.

  rebuild

    Same as build, but overwrites index.html, even if it was modified.

  serve

    Serve the files of a build or rebuild locally on port 8080. Opens
    a web browser at the local URL.

  run

    Does not generate any files in the current directory. Instead it
    builds main.wasm in a temporary folder and serves the template
    index.html file locally on port 8080. Opens a web browser at the
    local URL.

  help

    Displays this help text.`)
}

func main() {
	args := os.Args[1:]
	if len(args) == 1 && (args[0] == "build" || args[0] == "rebuild") {
		check(build(args[0] == "rebuild"))
	} else if len(args) == 1 && args[0] == "serve" {
		check(serve())
	} else if len(args) == 1 && args[0] == "run" {
		check(run())
	} else if len(args) == 0 || len(args) == 1 && args[0] == "help" {
		help()
	} else {
		fmt.Println("error: unknown command:", strings.Join(args, " "))
		help()
	}
}

func check(err error) {
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func build(rebuildTemplate bool) error {
	// Copy the Go installation's wasm_exec.js file which is needed to run the
	// build output.
	output, err := exec.Command("go", "env", "GOROOT").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to locate GOROOT: %w", err)
	}
	goroot := strings.TrimSpace(string(output))
	wasmExecPath := filepath.Join(goroot, "misc", "wasm", "wasm_exec.js")
	err = copyFile(wasmExecPath, "wasm_exec.js")
	if err != nil {
		return fmt.Errorf("failed to copy wasm_exec.js: %w", err)
	}

	// Build the WASM project.
	build := exec.Command("go", "build", "-o", "main.wasm")
	build.Env = append(
		os.Environ(),
		"GOOS=js",
		"GOARCH=wasm",
	)
	buildOutput, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", buildOutput)
	}

	existingIndexHtml, err := os.ReadFile("index.html")
	if rebuildTemplate || err != nil {
		// Either the command was rebuild or the command was build and
		// index.html does not yet exist, so we create it.
		err := os.WriteFile("index.html", indexHTML, 0666)
		if err != nil {
			return fmt.Errorf("failed to generate index.html: %w", err)
		}
	} else {
		// index.html already exists. In case it matches what we would have
		// generated, everything is fine. But if it differs from our template,
		// we warn the user. An older generated file might be used or a
		// previously generated index.html was modified. In this case we give
		// the user a warning because the existing index.html might be
		// incompatible.
		if !bytes.Equal(existingIndexHtml, indexHTML) {
			fmt.Println("warning: index.html already exists and differs from template; to re-generate it use drawsm rebuild")
		}
	}

	return nil
}

func copyFile(from, to string) error {
	data, err := os.ReadFile(from)
	if err != nil {
		return err
	}

	return os.WriteFile(to, data, 0666)
}

func serve() error {
	var httpErr error
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		httpErr = http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))
		if httpErr != nil {
			stop <- nil
		}
	}()

	// Wait a short while to let the http server start (or fail) and then open
	// the localhost URL in the browser.
	go func() {
		time.Sleep(100 * time.Millisecond)
		if httpErr == nil {
			url := "http://localhost:8080"
			err := openURL(url)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	<-stop

	return httpErr
}

func run() error {
	// Create a temporary folder for the WASM build output.
	tempDir, err := os.MkdirTemp("", "drawsm_")
	if err != nil {
		return fmt.Errorf("failed to create temp build dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Read the Go installation's wasm_exec.js file which is needed to run the
	// build output.
	output, err := exec.Command("go", "env", "GOROOT").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to locate GOROOT: %w", err)
	}
	goroot := strings.TrimSpace(string(output))
	wasmExec, err := os.ReadFile(filepath.Join(goroot, "misc", "wasm", "wasm_exec.js"))
	if err != nil {
		return fmt.Errorf("failed to read wasm_exec.js: %w", err)
	}

	// Build the WASM project into the temporary directory and read the file for
	// later serving via HTTP.
	mainWasmPath := filepath.Join(tempDir, "main.wasm")
	build := exec.Command("go", "build", "-o", mainWasmPath)
	build.Env = append(
		os.Environ(),
		"GOOS=js",
		"GOARCH=wasm",
	)

	buildOutput, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s", buildOutput)
	}

	mainWasm, err := os.ReadFile(mainWasmPath)
	if err != nil {
		return fmt.Errorf("failed to read main.wasm: %w", err)
	}

	// We serve three files specially:
	// - index.html is served from our template code
	// - wasm_exec.js is served from the Go installation (read above)
	// - main.wasm, our build output, is served from the bytes read above
	// The rest of the files are served from the project directory itself. This
	// way, if there are images or other resources located in the project
	// folder, they are found by the file server.
	var httpErr error
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		httpErr = http.ListenAndServe(":8080", &handler{
			wasmExec:    wasmExec,
			mainWasm:    mainWasm,
			fileHandler: http.FileServer(http.Dir(".")),
		})
		if httpErr != nil {
			stop <- nil
		}
	}()

	// Wait a short while to let the http server start (or fail) and then open
	// the localhost URL in the browser.
	go func() {
		time.Sleep(100 * time.Millisecond)
		if httpErr == nil {
			url := "http://localhost:8080"
			err := openURL(url)
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	<-stop

	return httpErr
}

type handler struct {
	mainWasm    []byte
	wasmExec    []byte
	fileHandler http.Handler
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/" || r.URL.Path == "/index.html" {
		w.Write(indexHTML)
	} else if r.URL.Path == "/main.wasm" {
		w.Write(h.mainWasm)
	} else if r.URL.Path == "/wasm_exec.js" {
		w.Write(h.wasmExec)
	} else {
		h.fileHandler.ServeHTTP(w, r)
	}
}

func openURL(url string) error {
	switch runtime.GOOS {
	case "windows":
		return exec.Command("cmd", "/c", "start", url).Start()
	case "linux":
		return exec.Command("xdg-open", url).Start()
	case "darwin":
		return exec.Command("open", url).Start()
	default:
		fmt.Println("Please navigate to", url)
		return nil
	}
}

var indexHTML = []byte(`<html>
	<head>
		<style>
			body {
				display: flex;
				justify-content: center;
				align-items: center;
				height: 100vh;
				margin: 0;
			}
		</style>
	</head>
	<body>
		<canvas id="gameCanvas" width="800" height="600"></canvas>
		<script src="wasm_exec.js"></script>
		<script>
		  const go = new Go();
		  WebAssembly.instantiateStreaming(fetch("main.wasm"), go.importObject).then((result) => {
			go.run(result.instance);
		  });
		</script>
	</body>
</html>
`)
