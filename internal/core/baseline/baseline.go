package baseline

import (
	"encoding/xml"
	"io"
	"os"
	"strings"

	"github.com/adkd/adkd/internal/core/pathutil"
	"github.com/adkd/adkd/internal/core/types"
)

// SmellBaseline represents the detekt baseline XML structure.
type SmellBaseline struct {
	XMLName                  xml.Name `xml:"SmellBaseline"`
	ManuallySuppressedIssues []ID     `xml:"ManuallySuppressedIssues>ID"`
	CurrentIssues            []ID     `xml:"CurrentIssues>ID"`
}

type ID struct {
	Value string `xml:",chardata"`
}

// LoadBaseline parses a Detekt baseline.xml and returns a map of all suppressed IDs.
func LoadBaseline(path string) (map[string]bool, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	bytes, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var baseline SmellBaseline
	if err := xml.Unmarshal(bytes, &baseline); err != nil {
		return nil, err
	}

	suppressed := make(map[string]bool)
	// Two separate loops to avoid the `append` mutation risk on the first
	// slice's underlying array when its cap exceeds its len. Functionally
	// equivalent to `append(a, b...)` iteration but safer.
	for _, id := range baseline.ManuallySuppressedIssues {
		suppressed[id.Value] = true
	}
	for _, id := range baseline.CurrentIssues {
		suppressed[id.Value] = true
	}

	return suppressed, nil
}

// IsSuppressed is preserved as a backward-compatible wrapper for callers
// and tests that do not yet pass a project root. New callers should use
// IsSuppressedWithRoot directly.
func IsSuppressed(finding types.Finding, baselineIDs map[string]bool) bool {
	return IsSuppressedWithRoot(finding, baselineIDs, "")
}

// IsSuppressedWithRoot decides whether a finding is suppressed by any of
// the IDs loaded from a detekt baseline.xml. The projectRoot is used to
// absolutize relative paths so matching is robust against detekt SARIF
// emitting mixed absolute/relative paths and across Windows vs Unix runners.
//
// Convention for detekt baseline IDs:
//
//	"<RuleShortName>:<Path>:<SignatureOrEmpty>"
//
// Example: "UnusedImports:app/src/main/java/.../File.kt:import foo"
//
// Rules:
//   - The rule short name must match the finding's rule (after detekt's
//     full-class-name prefix is stripped).
//   - The path must match the finding's file path via exact match (after
//     normalization) OR boundary-aware suffix match via pathutil.SuffixMatch.
//     This prevents the round-1 substring-fragility bug where baseline
//     entries like "MyFile.kt" would falsely suppress findings in unrelated
//     files like "OtherMyFile.kt".
func IsSuppressedWithRoot(finding types.Finding, baselineIDs map[string]bool, projectRoot string) bool {
	// Detekt uses simple class names like "UnusedImports", but
	// finding.Rule may be qualified (e.g. "detekt.style.UnusedImports").
	if finding.Rule == "" {
		return false
	}
	parts := strings.Split(finding.Rule, ".")
	simpleRuleName := parts[len(parts)-1]

	normFindingFile := pathutil.NormalizePath(finding.File, projectRoot)

	for id := range baselineIDs {
		idParts := strings.SplitN(id, ":", 3)
		if len(idParts) < 2 {
			continue
		}
		baselineRule := idParts[0]
		baselinePath := idParts[1]

		if baselineRule != simpleRuleName {
			continue
		}

		// Normalize the baseline path. Detekt baseline.xml typically stores
		// paths relative to the git root, hence we join with projectRoot if
		// it's non-empty.
		normBaselinePath := pathutil.NormalizePath(baselinePath, projectRoot)

		// One-directional match: finding's normalized path may END with the
		// baseline's normalized path (e.g. finding is absolute, baseline is
		// relative). The symmetric direction was removed in round-2 task #5
		// because it over-suppresses: a short baseline like "MyFile.kt"
		// would otherwise match TWO different absolute findings both ending
		// in "/MyFile.kt" in different packages of the same project.
		//
		// Exact match OR SuffixMatch(finding, baseline) — that's it.
		if normFindingFile == normBaselinePath ||
			pathutil.SuffixMatch(normFindingFile, normBaselinePath) {
			return true
		}
	}

	return false
}
