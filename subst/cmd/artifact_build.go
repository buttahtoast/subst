package cmd

import (
	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func newArtifactBuildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "push",
		Short: "Push an artifact to a registry",
		Long: heredoc.Doc(`
			Run 'subst artifact push' prints the given configuration (based on configuration files and env)`),
		RunE: configurationcmd,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd
}
