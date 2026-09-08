//go:build js && wasm

package rpc

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

// Call invokes an RPC method on the backend server.
func Call(method string, req any, res any) error {
	payload, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to encode request: %w", err)
	}

	url := fmt.Sprintf("/__goks_rpc?method=%s", method)

	ch := make(chan []byte, 1)
	errCh := make(chan error, 1)

	var thenFunc, textThenFunc, errTextThenFunc, catchFunc js.Func

	textThenFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer textThenFunc.Release()
		ch <- []byte(args[0].String())
		return nil
	})

	errTextThenFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer errTextThenFunc.Release()
		errCh <- fmt.Errorf("rpc error: %s", args[0].String())
		return nil
	})

	thenFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer thenFunc.Release()
		resp := args[0]
		if !resp.Get("ok").Bool() {
			resp.Call("text").Call("then", errTextThenFunc)
			return nil
		}
		resp.Call("text").Call("then", textThenFunc)
		return nil
	})

	catchFunc = js.FuncOf(func(this js.Value, args []js.Value) any {
		defer catchFunc.Release()
		errCh <- fmt.Errorf("fetch failed: %s", args[0].Get("message").String())
		return nil
	})

	options := js.Global().Get("Object").New()
	options.Set("method", "POST")
	options.Set("body", string(payload))
	headers := js.Global().Get("Object").New()
	headers.Set("Content-Type", "application/json")
	options.Set("headers", headers)

	js.Global().Get("window").Call("fetch", url, options).
		Call("then", thenFunc).
		Call("catch", catchFunc)

	select {
	case body := <-ch:
		if res != nil {
			return json.Unmarshal(body, res)
		}
		return nil
	case err := <-errCh:
		return err
	}
}
