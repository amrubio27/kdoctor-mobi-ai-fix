// kdoctor init: bootstrapper project-aware.
package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/config"
)

func NewInitCmd() *cobra.Command {
	var force bool
	var withSkills bool
	var projectType string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap kdoctor in the current directory",
		Long: `kdoctor init detects the project type, generates kdoctor.config.yaml,
and creates a minimal detekt.yml with the V1 live rules enabled.

Use --type to override auto-detection, --with-skills to generate AI agent rules, and --force to overwrite existing files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, initFlags{force: force, withSkills: withSkills, projectType: projectType})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing kdoctor.config.yaml and detekt.yml")
	cmd.Flags().BoolVar(&withSkills, "with-skills", false, "generate AI agent rules in .agents/skills/kdoctor-compose/SKILL.md")
	cmd.Flags().StringVar(&projectType, "type", "", "project type (android, kmp, cmp, compose, jvm, gradle, plain); auto-detected by default")
	return cmd
}

type initFlags struct {
	force       bool
	withSkills  bool
	projectType string
}

func runInit(cmd *cobra.Command, f initFlags) error {
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getcwd: %w", err)
	}

	pt := f.projectType
	if pt == "" {
		pt = detectProjectType(wd)
		fmt.Fprintf(cmd.OutOrStdout(), "Detected project type: %s\n", pt)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Using project type: %s\n", pt)
	}

	created := []string{}

	// 1. Preflight: si kdoctor.config.yaml o detekt.yml ya existen y no se especificó --force, rechazar
	configPath := filepath.Join(wd, "kdoctor.config.yaml")
	detektPath := filepath.Join(wd, "detekt.yml")
	if !f.force {
		var conflicts []string
		if _, err := os.Stat(configPath); err == nil {
			conflicts = append(conflicts, configPath)
		}
		if _, err := os.Stat(detektPath); err == nil {
			conflicts = append(conflicts, detektPath)
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("%s already exists (use --force to overwrite)", strings.Join(conflicts, ", "))
		}
	}

	cfg := config.ForProjectType(pt)
	data, err := config.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", configPath, err)
	}
	created = append(created, configPath)

	if err := os.WriteFile(detektPath, []byte(detektTemplate()), 0644); err != nil {
		return fmt.Errorf("write %s: %w", detektPath, err)
	}
	created = append(created, detektPath)

	// 3. .gitignore
	gitignorePath := filepath.Join(wd, ".gitignore")
	gitignoreUpdated, err := updateGitignore(gitignorePath, f.force)
	if err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}
	if gitignoreUpdated {
		created = append(created, gitignorePath)
	}

	// 4. .agents/skills/kdoctor-compose/SKILL.md
	if f.withSkills || pt == "cmp" || pt == "compose" || pt == "android" || pt == "kmp" {
		skillDir := filepath.Join(wd, ".agents", "skills", "kdoctor-compose")
		if err := os.MkdirAll(skillDir, 0755); err == nil {
			skillPath := filepath.Join(skillDir, "SKILL.md")
			writeSkill := f.force
			if !writeSkill {
				_, err := os.Stat(skillPath)
				writeSkill = os.IsNotExist(err)
			}
			if writeSkill {
				if err := os.WriteFile(skillPath, []byte(composeSkillTemplate()), 0644); err == nil {
					created = append(created, skillPath)
				}
			}
		}
	}

	// 5. Summary
	fmt.Fprintln(cmd.OutOrStdout(), "\nCreated/updated:")
	for _, p := range created {
		fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", p)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\nNext steps:")
	fmt.Fprintln(cmd.OutOrStdout(), "  kdoctor scan                    # run a scan")
	fmt.Fprintln(cmd.OutOrStdout(), "  kdoctor scan --json             # output JSON")
	fmt.Fprintln(cmd.OutOrStdout(), "  kdoctor scan --diff main        # only new findings")

	return nil
}

// detectProjectType infiere el tipo de proyecto a partir de archivos típicos.
func detectProjectType(root string) string {
	dirExists := func(path string) bool {
		info, err := os.Stat(filepath.Join(root, path))
		return err == nil && info.IsDir()
	}

	// CMP: Compose Multiplatform project structure.
	if dirExists("composeApp") {
		return "cmp"
	}

	// KMP: commonMain or Kotlin Multiplatform plugin.
	if dirExists("commonMain") {
		return "kmp"
	}

	// Gradle files content.
	kmpHints := []string{"kotlin(\"multiplatform\")", "kotlin-multiplatform", "org.jetbrains.kotlin.multiplatform"}
	cmpHints := []string{"org.jetbrains.compose", "compose-multiplatform"}
	androidHints := []string{"com.android.application", "com.android.library"}

	gradleFiles := []string{"build.gradle.kts", "build.gradle", "settings.gradle.kts", "settings.gradle"}
	for _, gf := range gradleFiles {
		path := filepath.Join(root, gf)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		text := string(data)
		if containsAny(text, cmpHints) {
			return "cmp"
		}
		if containsAny(text, kmpHints) {
			return "kmp"
		}
		if containsAny(text, androidHints) {
			return "android"
		}
	}

	// Android manifest.
	if _, err := os.Stat(filepath.Join(root, "src", "main", "AndroidManifest.xml")); err == nil {
		return "android"
	}

	// Compose-only project (no Android/KMP markers).
	if hasFileWithSubstring(root, "Composable") {
		return "compose"
	}

	// JVM: src/main/kotlin directory.
	if dirExists("src/main/kotlin") {
		return "jvm"
	}

	// Generic Gradle project.
	for _, gf := range gradleFiles {
		if _, err := os.Stat(filepath.Join(root, gf)); err == nil {
			return "gradle"
		}
	}

	return "plain"
}

// hasFileWithSubstring walks root up to maxDepth and returns true if any
// regular file name contains sub.
func hasFileWithSubstring(root, sub string) bool {
	found := false
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), sub) {
			found = true
		}
		return nil
	})
	return found
}

func containsAny(s string, substrs []string) bool {
	for _, sub := range substrs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

func detektTemplate() string {
	return `# detekt.yml — generated by kdoctor init.
# WARNING: when you pass --config detekt.yml to detekt-cli, it REPLACES
# (does not merge) the bundled default config. Only rules listed here fire.

coroutines:
  active: true
  GlobalCoroutineUsage:
    active: true

style:
  active: true
  UnusedImports:
    active: true

naming:
  active: true
  FunctionNaming:
    active: true
    # Compose functions use PascalCase; detekt's default expects camelCase.
    ignoreAnnotated:
      - "Composable"

complexity:
  active: true
  TooManyFunctions:
    active: true
`
}

var kdoctorGitignoreEntries = []string{
	"# kdoctor",
	"kdoctor.exe",
	".kdoctor/",
	"report.json",
	"*.sarif",
}

// updateGitignore asegura que las entradas de kdoctor existan en .gitignore.
// Si force=true, reescribe la sección; de lo contrario, solo añade lo faltante.
func updateGitignore(path string, force bool) (bool, error) {
	entries := kdoctorGitignoreEntries
	marker := entries[0] // "# kdoctor"

	if _, err := os.Stat(path); os.IsNotExist(err) {
		content := strings.Join(entries, "\n") + "\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return false, err
		}
		return true, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	existing := string(data)

	// Si force y existe sección kdoctor, reemplazar; si no, añadir.
	if force && strings.Contains(existing, marker) {
		start := strings.Index(existing, marker)
		end := start + len(marker)
		// buscar hasta el siguiente salto de línea que preceda a otro bloque
		for end < len(existing) && existing[end] != '\n' {
			end++
		}
		// avanzar hasta antes de la siguiente sección (línea que empieza con #)
		for end < len(existing) {
			if existing[end] == '\n' {
				peek := end + 1
				if peek < len(existing) && existing[peek] == '#' {
					break
				}
			}
			end++
		}
		existing = existing[:start] + strings.Join(entries, "\n") + "\n" + existing[end:]
		if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
			return false, err
		}
		return true, nil
	}

	if !strings.Contains(existing, marker) {
		content := existing + "\n" + strings.Join(entries, "\n") + "\n"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil
}

func composeSkillTemplate() string {
	return `---
name: kdoctor-compose
description: Modern Jetpack Compose, ViewModel state management, and Kotlin K2 quality guidelines enforced by kdoctor.
---

# kdoctor Modern Compose & Kotlin Rules

When writing or refactoring Kotlin & Jetpack Compose code:

1. **Lifecycle-Aware State Collection**:
   - ALWAYS prefer ` + "`collectAsStateWithLifecycle()`" + ` over ` + "`collectAsState()`" + ` to avoid collecting flow emissions while in background state.

2. **Atomic ViewModel State Updates**:
   - ALWAYS use ` + "`_uiState.update { ... }`" + ` instead of ` + "`_uiState.value = ...`" + ` to prevent race conditions in asynchronous code.
   - ALWAYS expose UI state as read-only ` + "`val uiState: StateFlow<MyUiState> = _uiState.asStateFlow()`" + `.

3. **Compiler K2 Stability & Recomposition**:
   - Annotate UI state data classes with ` + "`@Immutable`" + ` or ` + "`@Stable`" + `.
   - Use immutable collections (e.g. ` + "`kotlinx.collections.immutable.ImmutableList`" + `) in Composable parameter signatures.

4. **Lazy Layout Keys**:
   - ALWAYS provide an explicit ` + "`key`" + ` parameter in ` + "`LazyColumn`" + ` / ` + "`LazyRow`" + ` ` + "`items()``" + ` lambdas to preserve item identity during recompositions.

5. **Modifier Hoisting & Conventions**:
   - Maintain ` + "`modifier: Modifier = Modifier`" + ` as the first optional parameter of any Composable function.
`
}
