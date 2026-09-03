package grader

import (
	"testing"

	"github.com/adkd/adkd/internal/core/types"
)

func TestEmptyIs100(t *testing.T) {
	score, sum := Score(nil)
	if score != 100 || sum.Total != 0 {
		t.Fatalf("expected 100/empty, got %d / %+v", score, sum)
	}
}

func TestClusterMultipliers(t *testing.T) {
	// Security error (weight 5.0 * 2.0 = 10.0 penalty) -> score 90
	secScore, _ := Score([]types.Finding{
		{Severity: types.SeverityError, Cluster: "security"},
	})
	if secScore != 90 {
		t.Errorf("expected 90 for Security error, got %d", secScore)
	}

	// UI/Clean-code error (weight 5.0 * 0.75 = 3.75 -> round 4) -> score 96
	uiScore, _ := Score([]types.Finding{
		{Severity: types.SeverityError, Cluster: "clean-code"},
	})
	if uiScore != 96 {
		t.Errorf("expected 96 for Clean-code error, got %d", uiScore)
	}
}

func TestNoCliffAt300Lines(t *testing.T) {
	finding := []types.Finding{
		{Severity: types.SeverityError, Cluster: "compose-performance"},
	}
	score299, _ := ScoreWithKLOC(finding, 299)
	score301, _ := ScoreWithKLOC(finding, 301)

	if score299 != score301 {
		t.Fatalf("cliff detected: score at 299 lines is %d, score at 301 lines is %d (should be equal)", score299, score301)
	}
}

func TestInfoPenaltyCappedAt10(t *testing.T) {
	// 50 Info findings on different files
	in := make([]types.Finding, 50)
	for i := range in {
		in[i] = types.Finding{
			Severity: types.SeverityInfo,
			Cluster:  "clean-code",
			File:     string(rune('A' + i)),
			Rule:     "ui-hardcoded-strings",
		}
	}
	score, sum := Score(in)
	// 50 Info findings should be capped at 10 pts max deduction -> score >= 90
	if score != 90 {
		t.Fatalf("expected score capped at 90 due to max info penalty of 10, got %d (sum=%+v)", score, sum)
	}
}

func TestCriticalErrorsNotDilutedByKLOC(t *testing.T) {
	// 5 critical security errors in a 100,000 line project across distinct security rules
	findings := []types.Finding{
		{Severity: types.SeverityError, Cluster: "security", File: "A.kt", Rule: "sec-rule-1"},
		{Severity: types.SeverityError, Cluster: "security", File: "B.kt", Rule: "sec-rule-2"},
		{Severity: types.SeverityError, Cluster: "security", File: "C.kt", Rule: "sec-rule-3"},
		{Severity: types.SeverityError, Cluster: "security", File: "D.kt", Rule: "sec-rule-4"},
		{Severity: types.SeverityError, Cluster: "security", File: "E.kt", Rule: "sec-rule-5"},
	}
	score100K, _ := ScoreWithKLOC(findings, 100000)
	// Each critical security error is 5.0 * 2.0 = 10 pts. Total = 50 pts deduction.
	// Since critical errors are not diluted by KLOC, score should be 50.
	if score100K != 50 {
		t.Fatalf("critical errors should not be diluted by KLOC: expected 50, got %d", score100K)
	}
}

func TestCriticalErrorCappedPerRule(t *testing.T) {
	// 10 occurrences of the exact same critical rule across different files
	// Without cap: 10 * 7.5 = 75 pts penalty
	// With 15.0 max cap per rule: penalty capped at 15.0 -> score 85
	findings := make([]types.Finding, 10)
	for i := 0; i < 10; i++ {
		findings[i] = types.Finding{
			Severity: types.SeverityError,
			Cluster:  "architecture",
			Rule:     "arch-presentation-depends-on-data",
			File:     string(rune('A'+i)) + ".kt",
		}
	}
	score, _ := Score(findings)
	if score != 85 {
		t.Fatalf("expected 85 due to 15.0 cap on single critical rule, got %d", score)
	}
}

func TestDiminishingReturnsPerFileRule(t *testing.T) {
	// 4 repeated warnings on the same file & rule
	findings := []types.Finding{
		{Severity: types.SeverityWarning, Cluster: "compose-performance", File: "Screen.kt", Rule: "compose-recomposition-optimizer"},
		{Severity: types.SeverityWarning, Cluster: "compose-performance", File: "Screen.kt", Rule: "compose-recomposition-optimizer"},
		{Severity: types.SeverityWarning, Cluster: "compose-performance", File: "Screen.kt", Rule: "compose-recomposition-optimizer"},
		{Severity: types.SeverityWarning, Cluster: "compose-performance", File: "Screen.kt", Rule: "compose-recomposition-optimizer"},
	}
	score, _ := Score(findings)
	// Base weight = 2.0 * 1.0 (compose cluster)
	// Match 1: 2.0 * 1.0 = 2.0
	// Match 2: 2.0 * 0.75 = 1.5
	// Match 3: 2.0 * 0.50 = 1.0
	// Match 4: 2.0 * 0.25 = 0.5
	// Total penalty = 5.0 -> score = 95
	if score != 95 {
		t.Fatalf("expected 95 with diminishing returns, got %d", score)
	}
}
