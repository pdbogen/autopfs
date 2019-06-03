//go:generate env GOARCH=wasm GOOS=js go build -o assets/js/autopfs.wasm ./js/
//go:generate gzip -f -9 assets/js/autopfs.wasm
//go:generate go run ./assets
package main
