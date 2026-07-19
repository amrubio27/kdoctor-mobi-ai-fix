// Package main es el entrypoint de adkd.
//
// adkd (Android / KMP / CMP Doctor AI Fix) es un CLI que escanea
// proyectos Kotlin/Multiplatform, asigna un Health Score 0-100 y,
// opcionalmente, aplica auto-fix con un LLM.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/cli"
)

var version = "0.1.0"

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "adkd",
		Short: "Android / KMP / CMP Doctor with AI-driven fixes",
		Long: `adkd escanea tu proyecto Android, Kotlin Multiplatform o Compose
Multiplatform, calcula un Health Score 0-100 sobre un catálogo de
reglas de calidad y, opcionalmente, aplica auto-fix con un LLM
(Claude Code, Cursor, Gemini CLI, MobiAI).

Inspirado en react-doctor, alineado con MobiAI.`,
		Version: version,
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
