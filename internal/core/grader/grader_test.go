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

func TestThreeErrorsDrop15(t *testing.T) {
	score, _ := Score([]types.Finding{
		{Severity: types.SeverityError},
		{Severity: types.SeverityError},
		{Severity: types.SeverityError},
	})
	if score != 85 {
		t.Fatalf("expected 85, got %d", score)
	}
}

func TestClampedInfo(t *testing.T) {
	// 200 info = 200 * 0.5 = 100 deduction → clamp a 0
	in := make([]types.Finding, 200)
	for i := range in {
		in[i].Severity = types.SeverityInfo
	}
	score, sum := Score(in)
	if score != 0 {
		t.Fatalf("expected clamp to 0, got %d (sum=%+v)", score, sum)
	}
	if sum.Info != 200 {
		t.Fatalf("info count off: %d", sum.Info)
	}
}

func TestClampedErrors(t *testing.T) {
	// 50 errors → 250 deduction → clamp a 0, no negativo
	in := make([]types.Finding, 50)
	for i := range in {
		in[i].Severity = types.SeverityError
	}
	score, _ := Score(in)
	if score != 0 {
		t.Fatalf("expected clamp to 0, got %d", score)
	}
}

func TestMixedSummary(t *testing.T) {
	in := []types.Finding{
		{Severity: types.SeverityError},
		{Severity: types.SeverityWarning},
		{Severity: types.SeverityInfo},
		{Severity: types.SeverityInfo},
	}
	score, sum := Score(in)
	// 100 - 1*5 - 1*2 - 2*0.5(=1) = 92
	if score != 92 {
		t.Fatalf("expected 92, got %d", score)
	}
	if sum.Errors != 1 || sum.Warnings != 1 || sum.Info != 2 || sum.Total != 4 {
		t.Fatalf("summary off: %+v", sum)
	}
}
