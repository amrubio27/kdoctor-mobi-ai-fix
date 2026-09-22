// Package main es el entrypoint de kdoctor.
//
// kdoctor (Android / KMP / CMP Doctor AI Fix) es un CLI que escanea
// proyectos Kotlin/Multiplatform, asigna un Health Score 0-100 y,
// opcionalmente, aplica auto-fix con un LLM.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/amrubio27/kdoctor-mobi-ai-fix/internal/cli"
)

// version is injected at build time with
//
//	-ldflags "-X main.version=1.2.3"
//
// It used to be a hardcoded literal, so a binary cut from a later tag still
// claimed to be 0.6.0 and could not be correlated with the CHANGELOG.
var version = "dev"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kdoctor",
		Short: "Android / KMP / CMP Doctor with AI-driven fixes",
		Long: `kdoctor escanea tu proyecto Android, Kotlin Multiplatform o Compose
Multiplatform, calcula un Health Score 0-100 sobre un catálogo de
reglas de calidad y emite un plan de remediación para que lo aplique
tu agente (Claude Code, Cursor, MobiAI). kdoctor no llama a ningún
modelo ni edita tus ficheros.

Inspirado en react-doctor, alineado con MobiAI.`,
		Version: version,
		// A scan that fails its quality gate, or a detekt that will not run, is
		// not a usage error. Dumping the flag list after every such failure
		// buried the actual message, which matters most in CI logs.
		SilenceUsage: true,
		// main prints the error itself, so let it do that once.
		SilenceErrors: true,
	}
	root.AddCommand(cli.NewScanCmd())
	root.AddCommand(cli.NewFixCmd())
	root.AddCommand(cli.NewInitCmd())
	root.AddCommand(cli.NewRulesCmd())
	root.AddCommand(cli.NewDoctorCmd())
	return root
}

func main() {
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
