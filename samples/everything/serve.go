//go:build exclude

package main

import (
	"fmt"
	"net/http"
)

func main() {
	fmt.Println("serving...")
	http.ListenAndServe(":8080", http.FileServer(http.Dir(".")))
}
