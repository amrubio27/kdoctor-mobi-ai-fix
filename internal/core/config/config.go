// Package config carga y serializa kdoctor.config.yaml.
//
// Schema mínimo Fase 1:
//
//	projectType: android|kmp|cmp
//	paths: { kotlin: [...], compose: [...] }
//	rules: { "compose-remember-missing": "error" | "off" | "warn" }
//	score: { failBelow: 80 }
//	aiFixer: { provider: auto|mobiai|claude|cursor|gemini, mode: suggest|interactive|auto }
package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type RuleConf struct {
	Severity string                 `yaml:"severity"`
	Options  map[string]interface{} `yaml:"options,omitempty"`
}

type Config struct {
	ProjectType string              `yaml:"projectType"`
	Paths       map[string][]string `yaml:"paths"`
	Excludes    []string            `yaml:"excludes,omitempty"`
	Rules       map[string]string   `yaml:"rules"` // ruleID or clusterID -> severity ("off" disables)
	Score       struct {
		FailBelow int `yaml:"failBelow"`
	} `yaml:"score"`
	AiFixer struct {
		Provider string `yaml:"provider"`
		Mode     string `yaml:"mode"`
	} `yaml:"aiFixer"`
}

// Default devuelve una config sensata para proyectos Android.
func Default() Config {
	return ForProjectType("android")
}

// ForProjectType devuelve una configuración por defecto acorde al tipo de
// proyecto detectado. Si projectType no es reconocido, cae a "plain".
func ForProjectType(projectType string) Config {
	c := Config{ProjectType: projectType}
	switch projectType {
	case "kmp":
		c.Paths = map[string][]string{
			"kotlin":  {"commonMain/**/*.kt", "androidMain/**/*.kt", "jvmMain/**/*.kt", "**/*.kt"},
			"compose": {"commonMain/**/compose/*.kt", "commonMain/**/*Composable*.kt"},
		}
	case "cmp":
		c.Paths = map[string][]string{
			"kotlin":  {"composeApp/**/*.kt", "shared/**/*.kt", "**/*.kt"},
			"compose": {"composeApp/**/ui/**/*.kt", "shared/**/*Composable*.kt"},
		}
	case "android":
		c.Paths = map[string][]string{
			"kotlin":  {"app/src/main/**/*.kt", "**/*.kt"},
			"compose": {"**/*Composable*.kt"},
		}
	case "compose":
		c.Paths = map[string][]string{
			"kotlin":  {"**/*.kt"},
			"compose": {"**/*Composable*.kt"},
		}
	case "jvm":
		c.Paths = map[string][]string{
			"kotlin": {"src/main/**/*.kt", "**/*.kt"},
		}
	case "gradle":
		c.Paths = map[string][]string{
			"kotlin": {"**/*.kt"},
		}
	default:
		c.ProjectType = "plain"
		c.Paths = map[string][]string{
			"kotlin": {"**/*.kt"},
		}
	}
	c.Rules = map[string]string{}
	c.Excludes = []string{"**/build/**", "**/.gradle/**", "**/kspCaches/**"}
	c.Score.FailBelow = 80
	c.AiFixer.Provider = "auto"
	c.AiFixer.Mode = "suggest"
	return c
}

// Load lee el YAML en path. Si el fichero no existe, devuelve Default().
func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	if c.ProjectType == "" {
		c.ProjectType = "android"
	}
	if c.Score.FailBelow == 0 {
		c.Score.FailBelow = 80
	}
	if c.AiFixer.Provider == "" {
		c.AiFixer.Provider = "auto"
	}
	if c.AiFixer.Mode == "" {
		c.AiFixer.Mode = "suggest"
	}
	return c, nil
}

// Marshal helper.
func Marshal(c Config) ([]byte, error) {
	return yaml.Marshal(c)
}
