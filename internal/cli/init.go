// kdoctor init: crea kdoctor.config.yaml en el CWD con defaults sensatos.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/adkd/adkd/internal/core/config"
)

func NewInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Create kdoctor.config.yaml in the current directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "kdoctor.config.yaml"
			if !force {
				if _, err := os.Stat(path); err == nil {
					return fmt.Errorf("%s ya existe; usa --force para sobrescribir", path)
				}
			}
			c := config.Default()
			data, err := config.Marshal(c)
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, data, 0644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\u2713 Created %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite existing kdoctor.config.yaml")
	return cmd
}
