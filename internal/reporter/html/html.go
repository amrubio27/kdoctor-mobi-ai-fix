// Package html renderiza el reporte de kdoctor como una página web HTML5 interactiva autocontenida.
package html

import (
	"encoding/json"
	"html/template"
	"io"

	"github.com/adkd/adkd/internal/core/types"
)

const htmlTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>kdoctor Report — {{.Report.ProjectType}}</title>
  <style>
    :root {
      --bg: #0f172a;
      --card-bg: #1e293b;
      --card-border: #334155;
      --text: #f8fafc;
      --text-muted: #94a3b8;
      --accent-red: #ef4444;
      --accent-yellow: #f59e0b;
      --accent-blue: #3b82f6;
      --accent-green: #10b981;
    }
    * { box-sizing: border-box; margin: 0; padding: 0; }
    body {
      font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      background-color: var(--bg);
      color: var(--text);
      line-height: 1.5;
      padding: 2rem;
    }
    .container { max-width: 1200px; margin: 0 auto; }
    header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 2rem;
      padding-bottom: 1rem;
      border-bottom: 1px solid var(--card-border);
    }
    .brand { font-size: 1.75rem; font-weight: 700; color: #60a5fa; }
    .subtitle { color: var(--text-muted); font-size: 0.9rem; }
    .metrics-grid {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
      gap: 1rem;
      margin-bottom: 2rem;
    }
    .card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.5rem;
    }
    .card-title { font-size: 0.875rem; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.05em; }
    .card-value { font-size: 2.25rem; font-weight: 700; margin-top: 0.5rem; }
    .score-green { color: var(--accent-green); }
    .score-yellow { color: var(--accent-yellow); }
    .score-red { color: var(--accent-red); }
    .filters {
      display: flex;
      gap: 0.5rem;
      margin-bottom: 1.5rem;
      flex-wrap: wrap;
    }
    .filter-btn {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      color: var(--text-muted);
      padding: 0.5rem 1rem;
      border-radius: 0.5rem;
      cursor: pointer;
      font-weight: 500;
      transition: all 0.2s;
    }
    .filter-btn.active, .filter-btn:hover {
      background: #334155;
      color: var(--text);
      border-color: #60a5fa;
    }
    .finding-card {
      background: var(--card-bg);
      border: 1px solid var(--card-border);
      border-radius: 0.75rem;
      padding: 1.25rem;
      margin-bottom: 1rem;
      border-left: 4px solid var(--card-border);
    }
    .finding-card.severity-error { border-left-color: var(--accent-red); }
    .finding-card.severity-warning { border-left-color: var(--accent-yellow); }
    .finding-card.severity-info { border-left-color: var(--accent-blue); }
    .finding-header { display: flex; justify-content: space-between; align-items: flex-start; gap: 1rem; }
    .finding-title { font-weight: 600; font-size: 1.05rem; }
    .badge {
      padding: 0.2rem 0.6rem;
      border-radius: 9999px;
      font-size: 0.75rem;
      font-weight: 700;
      text-transform: uppercase;
    }
    .badge-error { background: rgba(239, 68, 68, 0.2); color: var(--accent-red); }
    .badge-warning { background: rgba(245, 158, 11, 0.2); color: var(--accent-yellow); }
    .badge-info { background: rgba(59, 130, 246, 0.2); color: var(--accent-blue); }
    .file-location {
      font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
      color: var(--text-muted);
      font-size: 0.875rem;
      margin: 0.4rem 0;
      word-break: break-all;
    }
    .fix-hint {
      background: rgba(16, 185, 129, 0.1);
      border: 1px solid rgba(16, 185, 129, 0.3);
      color: #6ee7b7;
      padding: 0.75rem;
      border-radius: 0.5rem;
      margin-top: 0.75rem;
      font-size: 0.9rem;
    }
  </style>
</head>
<body>
  <div class="container">
    <header>
      <div>
        <div class="brand">kdoctor</div>
        <div class="subtitle">Quality & Architecture Audit Report</div>
      </div>
      <div>
        <span class="subtitle">Schema v{{.Report.SchemaVersion}}</span>
      </div>
    </header>

    <div class="metrics-grid">
      <div class="card">
        <div class="card-title">Health Score</div>
        <div class="card-value {{if ge .Report.HealthScore 90}}score-green{{else if ge .Report.HealthScore 70}}score-yellow{{else}}score-red{{end}}">
          {{.Report.HealthScore}} / 100
        </div>
      </div>
      <div class="card">
        <div class="card-title">Total Issues</div>
        <div class="card-value">{{.Report.Summary.Total}}</div>
      </div>
      <div class="card">
        <div class="card-title">Errors</div>
        <div class="card-value score-red">{{.Report.Summary.Errors}}</div>
      </div>
      <div class="card">
        <div class="card-title">Warnings</div>
        <div class="card-value score-yellow">{{.Report.Summary.Warnings}}</div>
      </div>
      <div class="card">
        <div class="card-title">Info</div>
        <div class="card-value" style="color: var(--accent-blue);">{{.Report.Summary.Info}}</div>
      </div>
    </div>

    <div class="filters" id="filter-container">
      <button class="filter-btn active" onclick="filterFindings('all')">All ({{.Report.Summary.Total}})</button>
      <button class="filter-btn" onclick="filterFindings('error')">Errors ({{.Report.Summary.Errors}})</button>
      <button class="filter-btn" onclick="filterFindings('warning')">Warnings ({{.Report.Summary.Warnings}})</button>
      <button class="filter-btn" onclick="filterFindings('info')">Info ({{.Report.Summary.Info}})</button>
    </div>

    <div id="findings-list">
      {{range .Report.Findings}}
      <div class="finding-card severity-{{.Severity}}" data-severity="{{.Severity}}" data-cluster="{{.Cluster}}">
        <div class="finding-header">
          <div class="finding-title">[{{.Cluster}}] {{.ID}}</div>
          <span class="badge badge-{{.Severity}}">{{.Severity}}</span>
        </div>
        <div class="file-location">📄 {{.File}}:{{.Line}}:{{.Column}}</div>
        <div>{{.Message}}</div>
        {{if .FixHint}}
        <div class="fix-hint">
          <strong>💡 Recommendation:</strong> {{.FixHint}}
        </div>
        {{end}}
      </div>
      {{end}}
    </div>
  </div>

  <script>
    function filterFindings(sev) {
      document.querySelectorAll('.filter-btn').forEach(btn => btn.classList.remove('active'));
      event.target.classList.add('active');

      document.querySelectorAll('.finding-card').forEach(card => {
        if (sev === 'all' || card.getAttribute('data-severity') === sev) {
          card.style.display = 'block';
        } else {
          card.style.display = 'none';
        }
      });
    }
  </script>
</body>
</html>
`

// RenderHTML escribe el reporte interactivo HTML en `w`.
func RenderHTML(r types.Report, w io.Writer) error {
	tmpl, err := template.New("htmlReport").Parse(htmlTemplate)
	if err != nil {
		return err
	}
	data := struct {
		Report types.Report
	}{
		Report: r,
	}
	return tmpl.Execute(w, data)
}

// RenderHTMLJSON es un helper de serialización segura.
func RenderHTMLJSON(r types.Report) (string, error) {
	b, err := json.Marshal(r)
	return string(b), err
}
