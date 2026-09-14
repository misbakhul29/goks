package component

import (
	"context"
	"time"
)

// SuspenseProps configures a Suspense boundary.
type SuspenseProps struct {
	// Fallback is rendered immediately while the asynchronous resolver is pending.
	Fallback *Node

	// Async is the asynchronous function that resolves the final component node.
	Async func(ctx context.Context) (*Node, error)

	// Timeout specifies the maximum duration to wait for resolution.
	// Defaults to 10 seconds if not specified.
	Timeout time.Duration
}

// SuspenseBoundary is a Renderable component that represents an asynchronous streaming boundary.
type SuspenseBoundary struct {
	Fallback *Node
	Async    func(ctx context.Context) (*Node, error)
	Timeout  time.Duration
}

// Render implements Renderable for SuspenseBoundary.
// In non-streaming contexts (such as static export or synchronous SSR),
// it renders the Fallback node.
func (s *SuspenseBoundary) Render() *Node {
	if s.Fallback != nil {
		return s.Fallback
	}
	return H("div", Props{"class": "goks-suspense-loading"}, Text("Loading..."))
}

// Suspense creates a new Suspense component node with a fallback skeleton and an asynchronous data resolver.
// In streaming SSR, the fallback is streamed immediately to the client, and the resolved component is
// streamed and progressively swapped in once the asynchronous work finishes.
//
// Example:
//
//	component.Suspense(component.SuspenseProps{
//	    Fallback: goks.H("div", goks.Props{"class": "animate-pulse"}, goks.Text("Loading user profile...")),
//	    Async: func(ctx context.Context) (*component.Node, error) {
//	        user, err := fetchUser(ctx)
//	        if err != nil { return nil, err }
//	        return UserCard(user), nil
//	    },
//	})
func Suspense(props SuspenseProps) *Node {
	timeout := props.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	boundary := &SuspenseBoundary{
		Fallback: props.Fallback,
		Async:    props.Async,
		Timeout:  timeout,
	}

	node := C(boundary)
	if node.Props == nil {
		node.Props = make(Props)
	}
	node.Props["__goks_suspense_boundary"] = boundary
	return node
}
