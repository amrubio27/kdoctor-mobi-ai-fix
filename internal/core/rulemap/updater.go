package rulemap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/adkd/adkd/internal/core/types"
)

const DefaultRulesURL = "https://raw.githubusercontent.com/amrubio27/kdoctor-mobi-ai-fix/main/rules/metadata.json"

// FetchLatestRules descarga el archivo metadata.json desde remoteURL (o KDOCTOR_RULES_URL o la URL por defecto),
// valida que contenga reglas válidas y lo guarda en el directorio de caché local (~/.kdoctor/rules/metadata.json).
func FetchLatestRules(remoteURL string) (int, string, error) {
	if remoteURL == "" {
		if envURL := os.Getenv("KDOCTOR_RULES_URL"); envURL != "" {
			remoteURL = envURL
		} else {
			remoteURL = DefaultRulesURL
		}
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", remoteURL, nil)
	if err != nil {
		return 0, "", fmt.Errorf("create request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", fmt.Errorf("http fetch failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, "", fmt.Errorf("remote returned HTTP status %d (%s)", resp.StatusCode, resp.Status)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, "", fmt.Errorf("read response body: %w", err)
	}

	var rules []types.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return 0, "", fmt.Errorf("invalid rules JSON format: %w", err)
	}

	if len(rules) == 0 {
		return 0, "", fmt.Errorf("remote rules file contains 0 rules")
	}

	cachePath, err := GetUserCachePath()
	if err != nil {
		return 0, "", fmt.Errorf("get user cache path: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		return 0, "", fmt.Errorf("create user cache dir: %w", err)
	}

	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return 0, "", fmt.Errorf("write rules cache %s: %w", cachePath, err)
	}

	return len(rules), cachePath, nil
}
