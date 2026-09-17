package action_test

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/action"
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

func TestAction_FormHelper(t *testing.T) {
	// Without secret
	action.SetSecret(nil)
	formNode := action.Form("submitComment", component.Props{"class": "space-y-4"},
		html.Input().Attr("name", "comment").Attr("type", "text"),
		html.Button("Submit").Attr("type", "submit"),
	)

	if formNode.Tag != "form" {
		t.Fatalf("expected tag form, got %s", formNode.Tag)
	}
	if formNode.Props["action"] != "/__goks_action?name=submitComment" {
		t.Fatalf("unexpected action: %v", formNode.Props["action"])
	}
	if formNode.Props["method"] != "POST" {
		t.Fatalf("unexpected method: %v", formNode.Props["method"])
	}
	if formNode.Props["class"] != "space-y-4" {
		t.Fatalf("unexpected class: %v", formNode.Props["class"])
	}

	// First child must be hidden _action input
	if len(formNode.Children) < 3 {
		t.Fatalf("expected at least 3 children, got %d", len(formNode.Children))
	}
	hiddenAction := formNode.Children[0]
	if hiddenAction.Props["name"] != "_action" || hiddenAction.Props["value"] != "submitComment" {
		t.Fatalf("unexpected hidden action node: %+v", hiddenAction.Props)
	}
}

func TestAction_CSRF_TokenValidation(t *testing.T) {
	secret := []byte("super-secret-key-1234567890123456")
	action.SetSecret(secret)
	defer action.SetSecret(nil)

	action.Register("secureTransfer", func(ctx *action.Context) (any, error) {
		return "transferred", nil
	})

	handler := action.Handler()

	// 1. Missing CSRF token when secret is set -> Forbidden
	reqNoToken := httptest.NewRequest(http.MethodPost, "/__goks_action?name=secureTransfer", strings.NewReader("_action=secureTransfer"))
	reqNoToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqNoToken.Header.Set("Origin", "http://localhost:3000")
	reqNoToken.Host = "localhost:3000"
	recNoToken := httptest.NewRecorder()
	handler.ServeHTTP(recNoToken, reqNoToken)

	if recNoToken.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for missing CSRF token, got %d", recNoToken.Code)
	}

	// 2. Invalid CSRF token -> Forbidden
	reqBadToken := httptest.NewRequest(http.MethodPost, "/__goks_action?name=secureTransfer", strings.NewReader("_action=secureTransfer&_csrf=invalidtoken"))
	reqBadToken.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqBadToken.Header.Set("Origin", "http://localhost:3000")
	reqBadToken.Host = "localhost:3000"
	recBadToken := httptest.NewRecorder()
	handler.ServeHTTP(recBadToken, reqBadToken)

	if recBadToken.Code != http.StatusForbidden {
		t.Fatalf("expected 403 Forbidden for invalid CSRF token, got %d", recBadToken.Code)
	}

	// 3. Valid CSRF token via Form Helper
	sessionKey := "sess_alice_123"
	formNode := action.Form("secureTransfer", component.Props{"sessionKey": sessionKey})
	var csrfVal string
	for _, ch := range formNode.Children {
		if ch.Props["name"] == "_csrf" {
			csrfVal = ch.Props["value"].(string)
			break
		}
	}
	if csrfVal == "" {
		t.Fatal("expected CSRF token to be generated in Form helper")
	}

	formBody := "_action=secureTransfer&_csrf=" + csrfVal + "&_session_key=" + sessionKey
	reqGood := httptest.NewRequest(http.MethodPost, "/__goks_action?name=secureTransfer", strings.NewReader(formBody))
	reqGood.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	reqGood.Header.Set("Origin", "http://localhost:3000")
	reqGood.Host = "localhost:3000"
	recGood := httptest.NewRecorder()
	handler.ServeHTTP(recGood, reqGood)

	if recGood.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other for valid token, got %d: %s", recGood.Code, recGood.Body.String())
	}
}

func TestAction_Multipart_And_FileUpload(t *testing.T) {
	action.SetSecret(nil)

	var recordedTitle string
	var recordedFileName string
	var recordedFileContent string

	action.Register("uploadDoc", func(ctx *action.Context) (any, error) {
		recordedTitle = ctx.Get("title")
		file, header, err := ctx.File("document")
		if err != nil {
			return nil, err
		}
		defer file.Close()
		recordedFileName = header.Filename
		content, _ := io.ReadAll(file)
		recordedFileContent = string(content)
		return "uploaded", nil
	})

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("_action", "uploadDoc")
	_ = writer.WriteField("title", "Project Specs")

	part, err := writer.CreateFormFile("document", "spec.txt")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	_, _ = part.Write([]byte("GoKS Architecture v1.4.0"))
	_ = writer.Close()

	req := httptest.NewRequest(http.MethodPost, "/__goks_action?name=uploadDoc", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Origin", "http://localhost:3000")
	req.Host = "localhost:3000"
	rec := httptest.NewRecorder()

	action.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected 303 See Other, got %d: %s", rec.Code, rec.Body.String())
	}
	if recordedTitle != "Project Specs" {
		t.Fatalf("expected title 'Project Specs', got %q", recordedTitle)
	}
	if recordedFileName != "spec.txt" {
		t.Fatalf("expected filename 'spec.txt', got %q", recordedFileName)
	}
	if recordedFileContent != "GoKS Architecture v1.4.0" {
		t.Fatalf("expected content 'GoKS Architecture v1.4.0', got %q", recordedFileContent)
	}
}
