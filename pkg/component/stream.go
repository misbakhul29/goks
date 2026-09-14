package component

import (
	"context"
	"fmt"
	"html"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"
)

var globalSuspenseSeq uint64

// StreamOptions configures streaming SSR behavior.
type StreamOptions struct {
	// ChunkTimeout is the default timeout for async resolvers if not specified on the boundary.
	DefaultTimeout time.Duration
}

// StreamResult contains telemetry about the completed stream.
type StreamResult struct {
	SuspenseCount int
	Errors        []error
}

type boundaryItem struct {
	id       string
	boundary *SuspenseBoundary
}

// RenderToStream renders a component Node tree to an io.Writer.
// The initial shell (with Suspense fallback skeletons) is flushed immediately.
// As asynchronous Suspense boundaries complete, replacement `<template>` chunks and
// DOM-swap scripts are progressively written and flushed to the client.
func RenderToStream(ctx context.Context, w io.Writer, flusher http.Flusher, node *Node, opts ...StreamOptions) (*StreamResult, error) {
	if node == nil {
		return &StreamResult{}, nil
	}

	var boundaries []boundaryItem
	var sb strings.Builder

	// Step 1: Render initial HTML shell while identifying Suspense boundaries
	renderInitialShell(node, &sb, &boundaries)

	if _, err := io.WriteString(w, sb.String()); err != nil {
		return nil, err
	}
	if flusher != nil {
		flusher.Flush()
	}

	result := &StreamResult{
		SuspenseCount: len(boundaries),
	}

	// If there are no asynchronous Suspense boundaries, we are done
	if len(boundaries) == 0 {
		return result, nil
	}

	// Step 2: Concurrently resolve Suspense boundaries
	type chunkResult struct {
		id   string
		html string
		err  error
	}

	chunkChan := make(chan chunkResult, len(boundaries))

	for _, item := range boundaries {
		go func(it boundaryItem) {
			b := it.boundary
			timeout := b.Timeout
			if timeout <= 0 {
				timeout = 10 * time.Second
			}

			tctx, tcancel := context.WithTimeout(ctx, timeout)
			defer tcancel()

			if b.Async == nil {
				chunkChan <- chunkResult{id: it.id, html: RenderToString(b.Fallback)}
				return
			}

			resolvedNode, err := b.Async(tctx)
			if err != nil {
				chunkChan <- chunkResult{id: it.id, err: err}
				return
			}

			chunkChan <- chunkResult{id: it.id, html: RenderToString(resolvedNode)}
		}(item)
	}

	// Step 3: Stream replacement chunks as they arrive
	completed := 0
	for completed < len(boundaries) {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case res := <-chunkChan:
			completed++
			var replacementScript string
			if res.err != nil {
				result.Errors = append(result.Errors, res.err)
				replacementScript = fmt.Sprintf(
					`<template id="goks-c-%s"><div class="goks-suspense-error" style="color:#ef4444;font-size:0.875rem;">Component error: %s</div></template>`+"\n"+
						`<script>(function(){var s=document.getElementById("%s"),c=document.getElementById("goks-c-%s");if(s&&c){s.replaceWith(c.content.cloneNode(true));c.remove();}})();</script>`+"\n",
					res.id, html.EscapeString(res.err.Error()), res.id, res.id,
				)
			} else {
				replacementScript = fmt.Sprintf(
					`<template id="goks-c-%s">%s</template>`+"\n"+
						`<script>(function(){var s=document.getElementById("%s"),c=document.getElementById("goks-c-%s");if(s&&c){s.replaceWith(c.content.cloneNode(true));c.remove();}})();</script>`+"\n",
					res.id, res.html, res.id, res.id,
				)
			}

			if _, err := io.WriteString(w, replacementScript); err != nil {
				return result, err
			}
			if flusher != nil {
				flusher.Flush()
			}
		}
	}

	return result, nil
}

func renderInitialShell(node *Node, sb *strings.Builder, boundaries *[]boundaryItem) {
	if node == nil {
		return
	}

	// Check if this node is a SuspenseBoundary
	if sbBoundary, ok := getSuspenseBoundary(node); ok {
		seq := atomic.AddUint64(&globalSuspenseSeq, 1)
		id := fmt.Sprintf("goks-s-%d", seq)
		*boundaries = append(*boundaries, boundaryItem{id: id, boundary: sbBoundary})

		sb.WriteString(fmt.Sprintf(`<div id="%s" data-goks-suspense="pending">`, id))
		if sbBoundary.Fallback != nil {
			sb.WriteString(RenderToString(sbBoundary.Fallback))
		} else {
			sb.WriteString(`<div class="goks-suspense-loading">Loading...</div>`)
		}
		sb.WriteString(`</div>`)
		return
	}

	switch node.Type {
	case NodeTypeText:
		sb.WriteString(html.EscapeString(node.Text))
	case NodeTypeFragment:
		for _, child := range node.Children {
			renderInitialShell(child, sb, boundaries)
		}
	case NodeTypeComponent:
		if node.Component != nil {
			renderInitialShell(node.Component.Render(), sb, boundaries)
		}
	default: // NodeTypeElement
		sb.WriteString("<")
		sb.WriteString(node.Tag)

		for k, v := range node.Props {
			if strings.HasPrefix(k, "on") || k == "innerHTML" || strings.HasPrefix(k, "__goks") {
				continue
			}
			if k == "className" {
				k = "class"
			}
			escapedVal := html.EscapeString(fmt.Sprintf("%v", v))
			sb.WriteString(fmt.Sprintf(` %s="%s"`, k, escapedVal))
		}
		sb.WriteString(">")

		switch node.Tag {
		case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
			return
		}

		if inner, ok := node.Props["innerHTML"]; ok {
			sb.WriteString(fmt.Sprintf("%v", inner))
		} else {
			for _, child := range node.Children {
				renderInitialShell(child, sb, boundaries)
			}
		}

		sb.WriteString("</")
		sb.WriteString(node.Tag)
		sb.WriteString(">")
	}
}

func getSuspenseBoundary(node *Node) (*SuspenseBoundary, bool) {
	if node == nil {
		return nil, false
	}
	if node.Props != nil {
		if b, ok := node.Props["__goks_suspense_boundary"].(*SuspenseBoundary); ok {
			return b, true
		}
	}
	if sb, ok := node.Component.(*SuspenseBoundary); ok {
		return sb, true
	}
	return nil, false
}
