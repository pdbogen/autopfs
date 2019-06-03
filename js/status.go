package main

import (
	"encoding/json"
	"github.com/dennwc/dom"
	"github.com/pdbogen/autopfs/types"
	"net/url"
	"syscall/js"
	"time"
)

func IsView() bool {
	return Param("view") != ""
}

func Location() *url.URL {
	href := dom.GetWindow().JSValue().Get("location").Get("href").String()
	wsUrl, err := url.Parse(href)
	if err != nil {
		panic("parsing URL " + href + ": " + err.Error())
	}
	return wsUrl
}

func Param(name string) string {
	wsUrl := Location()
	values, err := url.ParseQuery(wsUrl.RawQuery)
	if err != nil {
		panic("parsing raw query " + wsUrl.RawQuery + ": " + err.Error())
	}

	return values.Get(name)
}

func WsUrl() *url.URL {
	href := dom.GetWindow().JSValue().Get("location").Get("href").String()
	wsUrl, err := url.Parse(href)
	if err != nil {
		panic("parsing URL " + href + ": " + err.Error())
	}
	if wsUrl.Scheme == "https" {
		wsUrl.Scheme = "wss"
	} else {
		wsUrl.Scheme = "ws"
	}
	v, err := url.ParseQuery(wsUrl.RawQuery)
	if err != nil {
		panic("parsing query " + wsUrl.RawQuery + ": " + err.Error())
	}
	wsUrl.Path += "/ws"
	wsUrl.RawQuery = v.Encode()
	return wsUrl
}

func Status() {
	since := time.Time{}
	wsUrl := WsUrl()

	messageList := js.Global().Get("document").Call("getElementById", "messageList")
	if messageList.Type().String() == "null" {
		panic("could not find messageList?!")
	}

	jobState := js.Global().Get("document").Call("getElementById", "jobState")
	if jobState.Type().String() == "null" {
		panic("could not find jobState?!")
	}

outer:
	for {
		ch := make(chan types.JobMessage)
		ws := js.Global().Get("WebSocket").New(wsUrl.String())
		ws.Call("addEventListener", "message", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) < 1 {
				println("got message event with no arguments")
				return nil
			}

			var msg types.JobMessage
			msgStr := args[0].Get("data").String()
			if err := json.Unmarshal([]byte(msgStr), &msg); err != nil {
				println("could not unmarshal message: " + msgStr)
				return nil
			}
			ch <- msg
			return nil
		}))
		ws.Call("addEventListener", "close", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			close(ch)
			return nil
		}))
		ws.Call("addEventListener", "error", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			close(ch)
			return nil
		}))
		for message := range ch {
			if message.Time.Before(since) {
				continue
			}

			pre := js.Global().Get("document").Call("createElement", "pre")
			pre.Set("textContent", message.Time.Format(time.RFC3339)+": "+message.Message)
			li := js.Global().Get("document").Call("createElement", "li")
			li.Call("appendChild", pre)
			messageList.Call("appendChild", li)

			if message.State != jobState.Get("textContent").String() {
				jobState.Set("textContent", message.State)
			}

			if message.Time.After(since) {
				since = message.Time
			}

			if message.State == "done" {
				break outer
			}
		}
		time.Sleep(time.Second)
	}

	if IsView() {
		return
	}

	js.Global().Get("document").Set("location", "/html?id="+Param("id"))
}
