package rules

import (
	"reflect"
	"strings"
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestStripComments(t *testing.T) {
	input := `
		// line comment
		val x = 10 /* block comment */
		val s = "string with // comments inside"
	`
	got := stripComments(input)
	if !containsWord(got, "x") {
		t.Error("should preserve code variables")
	}
	if stringsContainsAny(got, "line comment", "block comment") {
		t.Error("should strip comments")
	}
	if !stringsContainsAny(got, "string with // comments inside") {
		t.Error("should preserve string content")
	}
}

func TestStripCommentsAndStrings(t *testing.T) {
	input := `
		// line comment
		val x = "string literal" /* block comment */
	`
	got := stripCommentsAndStrings(input)
	if !containsWord(got, "x") {
		t.Error("should preserve code variables")
	}
	if stringsContainsAny(got, "line comment", "string literal", "block comment") {
		t.Error("should strip comments and string literals")
	}
}

func TestExtractArguments(t *testing.T) {
	cases := []struct {
		input    string
		keyword  string
		expected string
	}{
		{
			input:    `items(list, key = { it.id }) { ... }`,
			keyword:  "items",
			expected: "list, key = { it.id }",
		},
		{
			input:    `Log.d("Tag", "msg: " + email)`,
			keyword:  "Log.d",
			expected: `"Tag", "msg: " + email`,
		},
		{
			input:    `itemsIndexed(list, key = { _, item -> item.id })`,
			keyword:  "itemsIndexed",
			expected: "list, key = { _, item -> item.id }",
		},
	}

	for _, tc := range cases {
		idx := findKeywordIndex(tc.input, tc.keyword)
		if idx == -1 {
			t.Fatalf("keyword %s not found in input %s", tc.keyword, tc.input)
		}
		args, endIdx := extractArguments(tc.input, idx+len(tc.keyword))
		if endIdx == -1 {
			t.Fatalf("could not extract arguments for %s", tc.keyword)
		}
		if args != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, args)
		}
	}
}

func TestComposeMissingKeyDetector(t *testing.T) {
	rule := types.Rule{ID: "compose-missing-key", Cluster: "compose-performance", Severity: types.SeverityError}
	det := &ComposeMissingKeyDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name: "items without key",
			content: `
				LazyColumn {
					items(itemsList) { item -> Text(item.name) }
				}
			`,
			want: 1,
		},
		{
			name: "items with key parameter",
			content: `
				LazyColumn {
					items(itemsList, key = { it.id }) { item -> Text(item.name) }
				}
			`,
			want: 0,
		},
		{
			name: "itemsIndexed without key",
			content: `
				LazyColumn {
					itemsIndexed(itemsList) { idx, item -> Text(item.name) }
				}
			`,
			want: 1,
		},
		{
			name: "itemsIndexed with key",
			content: `
				LazyColumn {
					itemsIndexed(itemsList, key = { idx, item -> item.id }) { idx, item -> Text(item.name) }
				}
			`,
			want: 0,
		},
		{
			name: "items inside comments should be ignored",
			content: `
				// items(itemsList) { ... }
				/* items(itemsList) */
			`,
			want: 0,
		},
		{
			name: "items inside string literal should be ignored",
			content: `
				val s = "items(itemsList) should be checked"
			`,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestSecLogPiiDetector(t *testing.T) {
	rule := types.Rule{ID: "sec-log-pii", Cluster: "security", Severity: types.SeverityError}
	det := &SecLogPiiDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "logging email address",
			content: `Log.d("User", "user email: admin@domain.com")`,
			want:    1,
		},
		{
			name:    "logging password variable",
			content: `Log.w(TAG, "password is: $password")`,
			want:    1,
		},
		{
			name:    "logging token key",
			content: `Timber.e(t, "Auth token expired: $token")`,
			want:    1,
		},
		{
			name:    "harmless logging",
			content: `Log.i("Network", "Request completed successfully in 200ms")`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestSecWebViewJavascriptEnabledDetector(t *testing.T) {
	rule := types.Rule{ID: "sec-webview-javascript-enabled", Cluster: "security", Severity: types.SeverityError}
	det := &SecWebViewJavascriptEnabledDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "WebView enabling JS settings object",
			content: `webView.settings.javaScriptEnabled = true`,
			want:    1,
		},
		{
			name:    "WebView enabling JS direct settings",
			content: `settings.javaScriptEnabled = true`,
			want:    1,
		},
		{
			name:    "WebView setting JS to false",
			content: `settings.javaScriptEnabled = false`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

func TestCoroutineDispatchersHardcodedDetector(t *testing.T) {
	rule := types.Rule{ID: "coroutine-dispatchers-hardcoded", Cluster: "coroutines", Severity: types.SeverityInfo}
	det := &CoroutineDispatchersHardcodedDetector{rule: rule}

	tests := []struct {
		name    string
		content string
		want    int
	}{
		{
			name:    "hardcoded IO dispatcher",
			content: `withContext(Dispatchers.IO) { ... }`,
			want:    1,
		},
		{
			name:    "hardcoded Default dispatcher",
			content: `val scope = CoroutineScope(Dispatchers.Default)`,
			want:    1,
		},
		{
			name:    "injected dispatcher",
			content: `withContext(ioDispatcher) { ... }`,
			want:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commStr := stripComments(tt.content)
			commStringStr := stripCommentsAndStrings(tt.content)
			findings := det.Check("File.kt", tt.content, commStr, commStringStr)
			if len(findings) != tt.want {
				t.Errorf("Check() got %d findings, want %d", len(findings), tt.want)
			}
		})
	}
}

// ----------------------------------------------------
// Test Helpers
// ----------------------------------------------------

func stringsContainsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if reflect.ValueOf(sub).IsValid() && reflect.ValueOf(s).IsValid() {
			if strings.Contains(s, sub) {
				return true
			}
		}
	}
	return false
}

func findKeywordIndex(content, keyword string) int {
	return strings.Index(content, keyword)
}
