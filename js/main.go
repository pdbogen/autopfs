package main

import "syscall/js"

func main() {
	println("autopfs starting")
	switch js.Global().Get("Entrypoint").String() {
	case "Status":
		Status()
	case "Html":
		Html()
	}

	for {
		select {}
	}
}
