package grader

import (
	"fmt"
	"testing"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/core/types"
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

// TestCriticalErrorsResistDilutionByKLOC replaces an earlier test that
// required critical findings to be fully immune to project size.
//
// The argument for immunity was sound -- a PII leak is a leak whatever the
// project measures -- but the effect was that the score stopped answering
// "how much debt is there" and started answering "do you have two critical
// rules". On a real 16 KLOC project two capped rules pinned 30 points, 62% of
// the total penalty, regardless of the other 400 findings.
//
// Critical findings are now divided by sqrt(KLOC) while everything else is
// divided by KLOC, so they still dominate -- their relative weight grows like
// sqrt(size) -- without pinning the score.
func TestCriticalErrorsResistDilutionByKLOC(t *testing.T) {
	// Five distinct critical security rules, 10 pts each before normalisation.
	findings := []types.Finding{
		{Severity: types.SeverityError, Cluster: "security", File: "A.kt", Rule: "sec-rule-1"},
		{Severity: types.SeverityError, Cluster: "security", File: "B.kt", Rule: "sec-rule-2"},
		{Severity: types.SeverityError, Cluster: "security", File: "C.kt", Rule: "sec-rule-3"},
		{Severity: types.SeverityError, Cluster: "security", File: "D.kt", Rule: "sec-rule-4"},
		{Severity: types.SeverityError, Cluster: "security", File: "E.kt", Rule: "sec-rule-5"},
	}

	// Regular findings at the same count and size, for comparison.
	regular := make([]types.Finding, 0, 5)
	for i, f := range findings {
		regular = append(regular, types.Finding{
			Severity: types.SeverityError, Cluster: "complexity",
			File: f.File, Rule: fmt.Sprintf("cx-%d", i),
		})
	}

	const lines = 100000
	critScore, _ := ScoreWithKLOC(findings, lines)
	regScore, _ := ScoreWithKLOC(regular, lines)
	t.Logf("at %d lines: 5 critical -> %d, 5 regular -> %d", lines, critScore, regScore)

	// The point of "critical": they must cost meaningfully more than ordinary
	// findings of the same count and severity.
	if critScore >= regScore {
		t.Errorf("critical findings must outweigh regular ones: critical -> %d, regular -> %d",
			critScore, regScore)
	}

	// And they must not vanish into a large codebase.
	if critScore > 95 {
		t.Errorf("five critical security rules should still cost real points, got %d", critScore)
	}

	// A smaller project with the same critical findings must score worse:
	// same absolute debt, higher density.
	smallScore, _ := ScoreWithKLOC(findings, 4000)
	if smallScore >= critScore {
		t.Errorf("the same critical findings should hurt a small project more: "+
			"4 KLOC -> %d, 100 KLOC -> %d", smallScore, critScore)
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

// TestScoreIsInvariantToProjectSize pins the property the KLOC normalisation
// exists for and never had: the same density of problems must score the same
// regardless of how big the project is.
//
// The old sqrt(KLOC) divisor failed this badly. Findings grow roughly linearly
// with size, so dividing by the square root left a residue growing like
// sqrt(size): the same debt density scored 15 at 4 KLOC and 0 at 16, 60 and 200
// KLOC. Every mid-sized project hit the floor, which is why a real project
// reported 0/100 and the number stopped carrying information.
//
// TestNoCliffAt300Lines guarded against a discontinuity but said nothing about
// the shape of the curve, so this went unnoticed.
func TestScoreIsInvariantToProjectSize(t *testing.T) {
	// One finding per 400 lines of code, held constant across sizes.
	const linesPerFinding = 400

	makeFindings := func(n int) []types.Finding {
		out := make([]types.Finding, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, types.Finding{
				// Distinct files, so diminishing returns do not distort the ratio.
				File:     fmt.Sprintf("src/File%d.kt", i),
				Rule:     "complexity-long-method",
				ID:       "complexity-long-method",
				Cluster:  "complexity",
				Severity: types.SeverityWarning,
			})
		}
		return out
	}

	sizes := []int{4000, 16000, 60000, 200000}
	scores := make([]int, 0, len(sizes))
	for _, lines := range sizes {
		score, _ := ScoreWithKLOC(makeFindings(lines/linesPerFinding), lines)
		scores = append(scores, score)
		t.Logf("%6d lines, %3d findings -> score %d", lines, lines/linesPerFinding, score)
	}

	for i := 1; i < len(scores); i++ {
		// A couple of points of drift from integer rounding is fine; a collapse
		// to the floor is not.
		if diff := scores[i] - scores[0]; diff > 2 || diff < -2 {
			t.Errorf("same debt density scores %d at %d lines but %d at %d lines; "+
				"the normalisation is not size-invariant",
				scores[0], sizes[0], scores[i], sizes[i])
		}
	}
}

// A project with twice the debt density must score worse than one with half,
// at the same size. Invariance to size is worthless if the score also stops
// responding to what it measures.
func TestScoreStillRespondsToDensity(t *testing.T) {
	const lines = 20000

	build := func(n int) []types.Finding {
		out := make([]types.Finding, 0, n)
		for i := 0; i < n; i++ {
			out = append(out, types.Finding{
				File:     fmt.Sprintf("src/File%d.kt", i),
				Rule:     "complexity-long-method",
				ID:       "complexity-long-method",
				Cluster:  "complexity",
				Severity: types.SeverityWarning,
			})
		}
		return out
	}

	clean, _ := ScoreWithKLOC(build(10), lines)
	dirty, _ := ScoreWithKLOC(build(200), lines)
	if clean <= dirty {
		t.Fatalf("denser debt must score worse: 10 findings -> %d, 200 findings -> %d", clean, dirty)
	}
	t.Logf("at %d lines: 10 findings -> %d, 200 findings -> %d", lines, clean, dirty)
}

// Small projects must not be flattered by dividing by a fraction of a KLOC.
func TestSmallProjectsAreNotInflatedByNormalisation(t *testing.T) {
	findings := []types.Finding{
		{File: "a.kt", Rule: "r", ID: "r", Cluster: "complexity", Severity: types.SeverityWarning},
		{File: "b.kt", Rule: "r", ID: "r", Cluster: "complexity", Severity: types.SeverityWarning},
	}
	tiny, _ := ScoreWithKLOC(findings, 120)
	small, _ := ScoreWithKLOC(findings, 900)
	if tiny != small {
		t.Errorf("below the 1 KLOC floor the divisor must not change the score: 120 lines -> %d, 900 lines -> %d",
			tiny, small)
	}
}
