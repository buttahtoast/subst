package cmd

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func newArtifactCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "artifact",
		Short: "Artifact interaction",
		Long: heredoc.Doc(`
			Run 'subst artifact' prints the given configuration (based on configuration files and env)`),
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	cmd.AddCommand(newArtifactBuildCmd())
	return cmd
}
