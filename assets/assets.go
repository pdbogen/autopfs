package main

import (
	"github.com/shurcooL/vfsgen"
	"net/http"
)

var fs = http.Dir("assets")

func main() {
	if err := vfsgen.Generate(fs, vfsgen.Options{Filename: "server/assets_vfsdata.go"}); err != nil {
		panic(err)
	}
}
