// kdoctor fix: stub para Fase 3 (AI Fixer con Quality-Focused Prompting).
//
// En Fase 3 recibiremos un `detektrunner.Report` y delegaremos al LLM provider
// elegido (auto-detect). Ver Plan Tarea 3.5.
package cli

import "github.com/spf13/cobra"

func NewFixCmd() *cobra.Command {
	var fixMode string
	cmd := &cobra.Command{
		Use:   "fix",
		Short: "Apply AI-driven fixes to detected findings (Fase 3)",
		Long: `kdoctor fix --ai mueve los findings detectados a un LLM (Claude Code,
Cursor, Gemini CLI, MobiAI) y aplica los patches. Options:
  --mode suggest      : genera fixes.md, no toca código (default seguro)
  --mode interactive  : pregunta por cada fix
  --mode auto         : aplica todo + valida con patch guard
Implementación completa: Fase 3 del plan.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.PrintErrln("\u26a0 kdoctor fix --ai todavía no implementado (Fase 3 del plan).")
			cmd.PrintErrf("Modo pedido: %s\n", fixMode)
			return nil
		},
	}
	cmd.Flags().StringVar(&fixMode, "mode", "suggest", "fix mode: suggest | interactive | auto")
	return cmd
}
