//go:build js && wasm

// Package ws - Client-side WebSocket wrapper for GoKS WASM apps.
package ws

import (
	"fmt"
	"syscall/js"
)

// ClientConn is a browser-side WebSocket connection.
type ClientConn struct {
	ws        js.Value
	onMessage func(string)
	onClose   func()
	onOpen    func()
	onError   func(string)
}

// NewClient opens a WebSocket connection to the given URL.
//
//	conn := ws.NewClient("ws://localhost:3000/ws/chat")
//	conn.OnMessage(func(msg string) {
//	    fmt.Println("received:", msg)
//	})
func NewClient(url string) *ClientConn {
	c := &ClientConn{}
	wsObj := js.Global().Get("WebSocket").New(url)

	wsObj.Set("onmessage", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if c.onMessage != nil {
			c.onMessage(args[0].Get("data").String())
		}
		return nil
	}))

	wsObj.Set("onclose", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		if c.onClose != nil {
			c.onClose()
		}
		return nil
	}))

	wsObj.Set("onopen", js.FuncOf(func(_ js.Value, _ []js.Value) any {
		if c.onOpen != nil {
			c.onOpen()
		}
		return nil
	}))

	wsObj.Set("onerror", js.FuncOf(func(_ js.Value, args []js.Value) any {
		if c.onError != nil {
			c.onError(fmt.Sprintf("WebSocket error: %v", args[0]))
		}
		return nil
	}))

	c.ws = wsObj
	return c
}

// OnMessage registers a handler called when a message is received.
func (c *ClientConn) OnMessage(fn func(string)) { c.onMessage = fn }

// OnClose registers a handler called when the connection closes.
func (c *ClientConn) OnClose(fn func()) { c.onClose = fn }

// OnOpen registers a handler called when the connection opens.
func (c *ClientConn) OnOpen(fn func()) { c.onOpen = fn }

// OnError registers a handler called on WebSocket errors.
func (c *ClientConn) OnError(fn func(string)) { c.onError = fn }

// Send sends a text message to the server.
func (c *ClientConn) Send(msg string) {
	c.ws.Call("send", msg)
}

// Close closes the WebSocket connection.
func (c *ClientConn) Close() {
	c.ws.Call("close")
}

// ReadyState returns the WebSocket readyState value.
// 0=CONNECTING, 1=OPEN, 2=CLOSING, 3=CLOSED
func (c *ClientConn) ReadyState() int {
	return c.ws.Get("readyState").Int()
}
