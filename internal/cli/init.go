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
	var projectType string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap kdoctor in the current directory",
		Long: `kdoctor init detects the project type, generates kdoctor.config.yaml,
and creates a minimal detekt.yml with the V1 live rules enabled.

Use --type to override auto-detection and --force to overwrite existing files.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(cmd, initFlags{force: force, projectType: projectType})
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing kdoctor.config.yaml and detekt.yml")
	cmd.Flags().StringVar(&projectType, "type", "", "project type (android, kmp, cmp, compose, jvm, gradle, plain); auto-detected by default")
	return cmd
}

type initFlags struct {
	force       bool
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

	// 1. kdoctor.config.yaml
	configPath := filepath.Join(wd, "kdoctor.config.yaml")
	if !f.force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s already exists; use --force to overwrite", configPath)
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

	// 2. detekt.yml
	detektPath := filepath.Join(wd, "detekt.yml")
	writeDetekt := f.force
	if !writeDetekt {
		_, err := os.Stat(detektPath)
		writeDetekt = os.IsNotExist(err)
	}
	if writeDetekt || f.force {
		if err := os.WriteFile(detektPath, []byte(detektTemplate()), 0644); err != nil {
			return fmt.Errorf("write %s: %w", detektPath, err)
		}
		created = append(created, detektPath)
	}

	// 3. .gitignore
	gitignorePath := filepath.Join(wd, ".gitignore")
	gitignoreUpdated, err := updateGitignore(gitignorePath, f.force)
	if err != nil {
		return fmt.Errorf("update .gitignore: %w", err)
	}
	if gitignoreUpdated {
		created = append(created, gitignorePath)
	}

	// 4. Summary
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
