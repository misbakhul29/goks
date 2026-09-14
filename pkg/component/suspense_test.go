package component

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

type mockResponseWriter struct {
	bytes.Buffer
	header     http.Header
	flushed    int
	statusCode int
}

func newMockResponseWriter() *mockResponseWriter {
	return &mockResponseWriter{
		header: make(http.Header),
	}
}

func (m *mockResponseWriter) Header() http.Header {
	return m.header
}

func (m *mockResponseWriter) WriteHeader(statusCode int) {
	m.statusCode = statusCode
}

func (m *mockResponseWriter) Flush() {
	m.flushed++
}

func TestSuspense_SynchronousRenderToString(t *testing.T) {
	suspense := Suspense(SuspenseProps{
		Fallback: H("div", Props{"class": "skeleton"}, Text("Loading data...")),
		Async: func(ctx context.Context) (*Node, error) {
			return H("div", nil, Text("Resolved Data")), nil
		},
	})

	tree := H("main", nil,
		H("h1", nil, Text("Dashboard")),
		suspense,
	)

	html := RenderToString(tree)
	if !strings.Contains(html, "Loading data...") {
		t.Fatalf("Expected fallback in synchronous RenderToString, got: %s", html)
	}
	if strings.Contains(html, "Resolved Data") {
		t.Errorf("Synchronous RenderToString should not resolve async boundary without streaming")
	}
}

func TestSuspense_RenderToStream_Success(t *testing.T) {
	w := newMockResponseWriter()

	tree := H("div", Props{"class": "container"},
		H("header", nil, Text("GoKS Streaming SSR")),
		Suspense(SuspenseProps{
			Fallback: H("div", Props{"class": "spinner"}, Text("Fetching user profile...")),
			Async: func(ctx context.Context) (*Node, error) {
				time.Sleep(10 * time.Millisecond)
				return H("div", Props{"class": "profile"}, Text("User: Alice")), nil
			},
		}),
	)

	ctx := context.Background()
	res, err := RenderToStream(ctx, w, w, tree)
	if err != nil {
		t.Fatalf("RenderToStream failed: %v", err)
	}

	if res.SuspenseCount != 1 {
		t.Errorf("Expected SuspenseCount 1, got %d", res.SuspenseCount)
	}
	if len(res.Errors) != 0 {
		t.Errorf("Expected 0 errors, got %d", len(res.Errors))
	}

	output := w.String()

	// 1. Initial shell contains fallback and data-goks-suspense="pending"
	if !strings.Contains(output, `data-goks-suspense="pending"`) {
		t.Errorf("Expected data-goks-suspense attribute in initial shell")
	}
	if !strings.Contains(output, "Fetching user profile...") {
		t.Errorf("Expected fallback in output")
	}

	// 2. Stream contains template replacement and script
	if !strings.Contains(output, "<template id=\"goks-c-") {
		t.Errorf("Expected replacement template chunk in output")
	}
	if !strings.Contains(output, "User: Alice") {
		t.Errorf("Expected resolved content 'User: Alice' in output")
	}
	if !strings.Contains(output, "replaceWith") {
		t.Errorf("Expected DOM swap script in output")
	}

	// 3. Verify Flusher was called for both initial shell and replacement chunk
	if w.flushed < 2 {
		t.Errorf("Expected at least 2 flush calls, got %d", w.flushed)
	}
}

func TestSuspense_RenderToStream_MultipleBoundaries(t *testing.T) {
	w := newMockResponseWriter()

	tree := H("div", nil,
		Suspense(SuspenseProps{
			Fallback: Text("Loading 1..."),
			Async: func(ctx context.Context) (*Node, error) {
				time.Sleep(5 * time.Millisecond)
				return Text("Item 1 Ready"), nil
			},
		}),
		Suspense(SuspenseProps{
			Fallback: Text("Loading 2..."),
			Async: func(ctx context.Context) (*Node, error) {
				time.Sleep(10 * time.Millisecond)
				return Text("Item 2 Ready"), nil
			},
		}),
	)

	res, err := RenderToStream(context.Background(), w, w, tree)
	if err != nil {
		t.Fatalf("RenderToStream failed: %v", err)
	}

	if res.SuspenseCount != 2 {
		t.Errorf("Expected 2 suspense boundaries, got %d", res.SuspenseCount)
	}

	output := w.String()
	if !strings.Contains(output, "Item 1 Ready") || !strings.Contains(output, "Item 2 Ready") {
		t.Errorf("Expected both boundaries to resolve, got: %s", output)
	}
}

func TestSuspense_RenderToStream_ErrorHandling(t *testing.T) {
	w := newMockResponseWriter()

	tree := H("div", nil,
		Suspense(SuspenseProps{
			Fallback: Text("Loading items..."),
			Async: func(ctx context.Context) (*Node, error) {
				return nil, errors.New("upstream service timed out")
			},
		}),
	)

	res, err := RenderToStream(context.Background(), w, w, tree)
	if err != nil {
		t.Fatalf("RenderToStream returned unexpected top-level error: %v", err)
	}

	if len(res.Errors) != 1 {
		t.Fatalf("Expected 1 error in StreamResult, got %d", len(res.Errors))
	}

	output := w.String()
	if !strings.Contains(output, "goks-suspense-error") {
		t.Errorf("Expected error template in output, got: %s", output)
	}
	if !strings.Contains(output, "upstream service timed out") {
		t.Errorf("Expected error message in streamed template")
	}
}

func TestSuspense_RenderToStream_ContextCancellation(t *testing.T) {
	w := newMockResponseWriter()

	ctx, cancel := context.WithCancel(context.Background())

	tree := H("div", nil,
		Suspense(SuspenseProps{
			Fallback: Text("Loading..."),
			Async: func(c context.Context) (*Node, error) {
				select {
				case <-c.Done():
					return nil, c.Err()
				case <-time.After(500 * time.Millisecond):
					return Text("Done"), nil
				}
			},
		}),
	)

	// Cancel context immediately after starting
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	_, err := RenderToStream(ctx, w, w, tree)
	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got: %v", err)
	}
}

func TestSuspense_XSS_Escaping(t *testing.T) {
	w := newMockResponseWriter()

	maliciousText := `<script>alert('xss')</script>`
	tree := H("div", nil,
		Suspense(SuspenseProps{
			Fallback: Text(maliciousText),
			Async: func(ctx context.Context) (*Node, error) {
				return Text(maliciousText), nil
			},
		}),
	)

	_, err := RenderToStream(context.Background(), w, w, tree)
	if err != nil {
		t.Fatalf("RenderToStream failed: %v", err)
	}

	output := w.String()
	if strings.Contains(output, "<script>alert('xss')</script>") {
		t.Errorf("Found unescaped XSS script tags in streamed output: %s", output)
	}
	if !strings.Contains(output, "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;") {
		t.Errorf("Expected escaped HTML special characters in output")
	}
}
