// adkd rules: lista todas las reglas del catálogo con status.
package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/rulemap"
)

func NewRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "List all rules in the catalog",
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := resolveRulesPath()
			if err != nil {
				return err
			}
			rules, err := rulemap.LoadRules(path)
			if err != nil {
				return err
			}
			sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
			live, planned := 0, 0
			for _, r := range rules {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s [%s] %s — %s\n",
					r.ID, r.Cluster, r.Severity, r.Status)
				if r.Status == "live" {
					live++
				} else {
					planned++
				}
			}
			fmt.Fprintf(cmd.OutOrStdout(),
				"\nTotal: %d rules (%d live, %d planned)\n",
				len(rules), live, planned)
			return nil
		},
	}
	return cmd
}
