package mobiai

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestUploadFindings_Success(t *testing.T) {
	var received []byte
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected application/json content-type")
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			t.Errorf("expected bearer token, got %q", auth)
		}
		received, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := New(server.URL, "test-token")
	findings := []types.Finding{
		{ID: "arch-god-class", File: "App.kt", Line: 10, Column: 5, Severity: "warning", Message: "too many functions"},
	}
	if err := client.UploadFindings(context.Background(), "/project", findings); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var payload uploadPayload
	if err := json.Unmarshal(received, &payload); err != nil {
		t.Fatalf("parse payload: %v", err)
	}
	if payload.Tool != "kdoctor" {
		t.Errorf("tool=%q, want kdoctor", payload.Tool)
	}
	if payload.ProjectPath != "/project" {
		t.Errorf("projectPath=%q, want /project", payload.ProjectPath)
	}
	if len(payload.Findings) != 1 {
		t.Fatalf("len(findings)=%d, want 1", len(payload.Findings))
	}
	if payload.Findings[0].RuleID != "arch-god-class" {
		t.Errorf("ruleID=%q, want arch-god-class", payload.Findings[0].RuleID)
	}
}

func TestUploadFindings_NoBaseURL(t *testing.T) {
	client := New("", "")
	findings := []types.Finding{{ID: "x"}}
	if err := client.UploadFindings(context.Background(), "/project", findings); err == nil {
		t.Fatal("expected error when BaseURL is empty")
	}
}

func TestUploadFindings_EmptyFindings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("server should not be called when there are no findings")
	}))
	defer server.Close()

	client := New(server.URL, "")
	if err := client.UploadFindings(context.Background(), "/project", nil); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUploadFindings_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("boom"))
	}))
	defer server.Close()

	client := New(server.URL, "")
	findings := []types.Finding{{ID: "x"}}
	if err := client.UploadFindings(context.Background(), "/project", findings); err == nil {
		t.Fatal("expected error on 500")
	}
}
