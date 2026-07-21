// Package mobiai implements a thin client for the MobiAI Graph API.
package mobiai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/adkd/adkd/internal/core/types"
)

// Client posts kdoctor findings to a MobiAI Graph-compatible endpoint.
type Client struct {
	BaseURL string
	Token   string
	HTTP    *http.Client
}

// New returns a client configured with the given endpoint and token.
func New(baseURL, token string) *Client {
	return &Client{
		BaseURL: baseURL,
		Token:   token,
		HTTP: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// UploadFindings sends the findings as a JSON array to the graph endpoint.
// If the client has no BaseURL, it returns an error.
func (c *Client) UploadFindings(ctx context.Context, projectPath string, findings []types.Finding) error {
	if c.BaseURL == "" {
		return fmt.Errorf("mobiai BaseURL is empty")
	}
	if len(findings) == 0 {
		return nil
	}

	payload := uploadPayload{
		Tool:        "kdoctor",
		ProjectPath: projectPath,
		Findings:    toAnnotations(findings),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := strings.TrimSuffix(c.BaseURL, "/")
	url += "/graph/findings"

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("post to %s: %w", url, err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("mobiai returned %d: %s", resp.StatusCode, string(respBody))
}

type uploadPayload struct {
	Tool        string       `json:"tool"`
	ProjectPath string       `json:"projectPath"`
	Findings    []annotation `json:"findings"`
}

type annotation struct {
	URI       string `json:"uri"`
	StartLine int    `json:"startLine"`
	StartCol  int    `json:"startColumn"`
	RuleID    string `json:"ruleId"`
	Severity  string `json:"severity"`
	Message   string `json:"message"`
}

func toAnnotations(findings []types.Finding) []annotation {
	out := make([]annotation, 0, len(findings))
	for _, f := range findings {
		out = append(out, annotation{
			URI:       f.File,
			StartLine: f.Line,
			StartCol:  f.Column,
			RuleID:    f.ID,
			Severity:  string(f.Severity),
			Message:   f.Message,
		})
	}
	return out
}
