package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
)

// Server implements a JSON-RPC 2.0 Language Server for GOX files.
type Server struct {
	mu        sync.RWMutex
	docs      map[string]string // URI -> file content
	isShutdown bool
}

// NewServer creates a new GOX Language Server instance.
func NewServer() *Server {
	return &Server{
		docs: make(map[string]string),
	}
}

// Serve reads JSON-RPC messages from in and writes responses to out.
func (s *Server) Serve(in io.Reader, out io.Writer) error {
	reader := bufio.NewReader(in)

	for {
		// Read headers
		contentLength := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}

			line = strings.TrimSpace(line)
			if line == "" {
				// Blank line indicates end of headers
				break
			}

			if strings.HasPrefix(strings.ToLower(line), "content-length:") {
				parts := strings.Split(line, ":")
				if len(parts) == 2 {
					if cl, err := strconv.Atoi(strings.TrimSpace(parts[1])); err == nil {
						contentLength = cl
					}
				}
			}
		}

		if contentLength == 0 {
			continue
		}

		// Read exact body length
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			return err
		}

		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			continue
		}

		s.handleMessage(&req, out)
	}
}

func (s *Server) handleMessage(req *Request, out io.Writer) {
	switch req.Method {
	case "initialize":
		res := InitializeResult{
			Capabilities: ServerCapabilities{
				TextDocumentSync: 1, // Full document sync
				CompletionProvider: &CompletionOptions{
					ResolveProvider:   false,
					TriggerCharacters: []string{"<", " ", "/", "\"", "{", "."},
				},
				HoverProvider:              true,
				DocumentFormattingProvider: true,
			},
			ServerInfo: ServerInfo{
				Name:    "goks-lsp",
				Version: "0.2.0",
			},
		}
		s.sendResponse(out, req.ID, res, nil)

	case "initialized":
		// Notification from client, nothing to respond

	case "shutdown":
		s.mu.Lock()
		s.isShutdown = true
		s.mu.Unlock()
		s.sendResponse(out, req.ID, nil, nil)

	case "exit":
		os.Exit(0)

	case "textDocument/didOpen":
		var params DidOpenTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.Lock()
			s.docs[params.TextDocument.URI] = params.TextDocument.Text
			s.mu.Unlock()

			s.publishDiagnostics(out, params.TextDocument.URI, params.TextDocument.Text)
		}

	case "textDocument/didChange":
		var params DidChangeTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			if len(params.ContentChanges) > 0 {
				newText := params.ContentChanges[len(params.ContentChanges)-1].Text
				s.mu.Lock()
				s.docs[params.TextDocument.URI] = newText
				s.mu.Unlock()

				s.publishDiagnostics(out, params.TextDocument.URI, newText)
			}
		}

	case "textDocument/didSave":
		var params DidSaveTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.RLock()
			content := s.docs[params.TextDocument.URI]
			s.mu.RUnlock()

			s.publishDiagnostics(out, params.TextDocument.URI, content)
		}

	case "textDocument/didClose":
		var params DidCloseTextDocumentParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.Lock()
			delete(s.docs, params.TextDocument.URI)
			s.mu.Unlock()
		}

	case "textDocument/completion":
		var params CompletionParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.RLock()
			content := s.docs[params.TextDocument.URI]
			s.mu.RUnlock()

			items := GetCompletions(content, params.Position, params.TextDocument.URI)
			s.sendResponse(out, req.ID, CompletionList{IsIncomplete: false, Items: items}, nil)
		} else {
			s.sendResponse(out, req.ID, CompletionList{Items: []CompletionItem{}}, nil)
		}

	case "textDocument/hover":
		var params HoverParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.RLock()
			content := s.docs[params.TextDocument.URI]
			s.mu.RUnlock()

			hover := GetHover(content, params.Position)
			s.sendResponse(out, req.ID, hover, nil)
		} else {
			s.sendResponse(out, req.ID, nil, nil)
		}

	case "textDocument/formatting":
		var params DocumentFormattingParams
		if err := json.Unmarshal(req.Params, &params); err == nil {
			s.mu.RLock()
			content := s.docs[params.TextDocument.URI]
			s.mu.RUnlock()

			edits := FormatDocument(content)
			s.sendResponse(out, req.ID, edits, nil)
		} else {
			s.sendResponse(out, req.ID, nil, nil)
		}

	default:
		if req.ID != nil {
			// Unknown request method, return error
			s.sendResponse(out, req.ID, nil, &ResponseError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			})
		}
	}
}

func (s *Server) publishDiagnostics(out io.Writer, uri, content string) {
	diagnostics := ValidateDocument(uri, content)
	if diagnostics == nil {
		diagnostics = []Diagnostic{}
	}
	params := PublishDiagnosticsParams{
		URI:         uri,
		Diagnostics: diagnostics,
	}

	notif := Notification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  params,
	}

	data, err := json.Marshal(notif)
	if err != nil {
		return
	}

	s.writeRawMessage(out, data)
}

func (s *Server) sendResponse(out io.Writer, id *json.RawMessage, result any, respErr *ResponseError) {
	if id == nil {
		return // Notification, no response needed
	}

	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
		Error:   respErr,
	}

	data, err := json.Marshal(resp)
	if err != nil {
		log.Printf("[LSP] Error marshaling response: %v", err)
		return
	}

	s.writeRawMessage(out, data)
}

func (s *Server) writeRawMessage(out io.Writer, payload []byte) {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	_, _ = out.Write([]byte(header))
	_, _ = out.Write(payload)
}
