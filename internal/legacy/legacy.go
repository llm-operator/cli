package legacy

import (
	"github.com/llm-operator/cli/internal/legacy/completions"
	"github.com/spf13/cobra"
)

// Cmd is the root command for legacy.
func Cmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:                "legacy",
		Short:              "Legacy commands",
		Args:               cobra.NoArgs,
		DisableFlagParsing: true,
	}
	cmd.AddCommand(completions.Cmd())
	return cmd
}