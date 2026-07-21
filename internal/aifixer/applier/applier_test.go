package applier

import "testing"

func TestExtractCodeBlock(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "simple kotlin fence",
			input: "```kotlin\nfun a() {}\n```",
			want:  "fun a() {}",
		},
		{
			name:  "no language tag",
			input: "```\nfun a() {}\n```",
			want:  "fun a() {}",
		},
		{
			name:  "leading prose",
			input: "Here is the fix:\n```kotlin\nfun a() {}\n```\nDone.",
			want:  "fun a() {}",
		},
		{
			name:  "plain code without fences",
			input: "fun a() {}",
			want:  "fun a() {}",
		},
		{
			name:    "empty response",
			input:   "   ",
			wantErr: true,
		},
		{
			name:    "missing closing fence",
			input:   "```kotlin\nfun a() {}",
			wantErr: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ExtractCodeBlock(c.input)
			if c.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != c.want {
				t.Fatalf("want %q, got %q", c.want, got)
			}
		})
	}
}

func TestApplyPatch(t *testing.T) {
	tests := []struct {
		name      string
		original  string
		snippet   string
		start     int
		end       int
		want      string
		wantError bool
	}{
		{
			name:     "replace middle lines",
			original: "line1\nline2\nline3\nline4\nline5\n",
			snippet:  "new2\nnew3",
			start:    1,
			end:      3,
			want:     "line1\nnew2\nnew3\nline4\nline5\n",
		},
		{
			name:     "replace first line",
			original: "line1\nline2\n",
			snippet:  "new1",
			start:    0,
			end:      1,
			want:     "new1\nline2\n",
		},
		{
			name:     "replace last line",
			original: "line1\nline2\n",
			snippet:  "new2",
			start:    1,
			end:      2,
			want:     "line1\nnew2\n",
		},
		{
			name:     "empty snippet removes lines",
			original: "line1\nline2\nline3\n",
			snippet:  "",
			start:    1,
			end:      2,
			want:     "line1\nline3\n",
		},
		{
			name:     "strips leaked line numbers",
			original: "line1\nline2\nline3\n",
			snippet:  "2: new2",
			start:    1,
			end:      2,
			want:     "line1\nnew2\nline3\n",
		},
		{
			name:     "partial snippet replaces whole window",
			original: "line1\nline2\nline3\nline4\n",
			snippet:  "only",
			start:    1,
			end:      3,
			want:     "line1\nonly\nline4\n",
		},
		{
			name:      "start out of range",
			original:  "line1\nline2",
			snippet:   "x",
			start:     5,
			end:       6,
			wantError: true,
		},
		{
			name:      "end before start",
			original:  "line1\nline2",
			snippet:   "x",
			start:     2,
			end:       1,
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ApplyPatch(tt.original, tt.snippet, tt.start, tt.end)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Fatalf("want %q, got %q", tt.want, got)
			}
		})
	}
}
