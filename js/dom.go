package main

import "syscall/js"

func CreateElement(tag string) js.Value {
	return js.Global().Get("document").Call("createElement", tag)
}

func CreateElementText(tag, text string) js.Value {
	e := CreateElement(tag)
	e.Set("textContent", text)
	return e
}

func CreateTextNode(text string) js.Value {
	return js.Global().Get("document").Call("createTextNode", text)
}
