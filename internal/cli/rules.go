package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/rulemap"
)

func NewRulesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage and inspect kdoctor rules catalog",
		Long:  `Inspect available rules or update the rules catalog from remote GitHub repository into local user cache.`,
	}

	cmd.AddCommand(newRulesUpdateCmd())
	cmd.AddCommand(newRulesListCmd())

	return cmd
}

func newRulesUpdateCmd() *cobra.Command {
	var url string

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update rules catalog from remote GitHub repository to local user cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Fetching latest rules catalog...")
			count, path, err := rulemap.FetchLatestRules(url)
			if err != nil {
				return fmt.Errorf("update failed: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Successfully updated %d rules into local cache:\n  ✓ %s\n", count, path)
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "custom remote URL for metadata.json")
	return cmd
}

func newRulesListCmd() *cobra.Command {
	var projectDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all loaded rules and their current source (cache, project, or embedded)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if projectDir == "" {
				projectDir, _ = os.Getwd()
			}
			res, err := rulemap.LoadRulesCascade(projectDir, "")
			if err != nil {
				return fmt.Errorf("load rules: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Rule source active: %s (%s)\n", res.Source, res.Path)
			fmt.Fprintf(cmd.OutOrStdout(), "Total rules loaded: %d\n\n", len(res.Rules))

			fmt.Fprintf(cmd.OutOrStdout(), "%-35s %-20s %-10s %-8s\n", "RULE ID", "CLUSTER", "SEVERITY", "STATUS")
			fmt.Fprintf(cmd.OutOrStdout(), "--------------------------------------------------------------------------------\n")
			for _, r := range res.Rules {
				fmt.Fprintf(cmd.OutOrStdout(), "%-35s %-20s %-10s %-8s\n", r.ID, r.Cluster, r.Severity, r.Status)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&projectDir, "project-dir", "", "project directory to check for local overrides")
	return cmd
}
