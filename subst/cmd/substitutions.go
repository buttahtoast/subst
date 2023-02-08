package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/subst"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func newSubstitutionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "substitutions",
		Short: "Render available substitutions",
		Long: heredoc.Doc(`
			Run 'subst substitutions' to return available substitutions for given Kustomize.`),
		RunE: substitutions,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd

}

func substitutions(cmd *cobra.Command, args []string) error {
	configuration, err := config.LoadConfiguration(cfgFile, cmd)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	m, err := subst.New(*configuration)
	if err != nil {
		return err
	}
	if m != nil {
		if len(m.Substitutions.Subst) > 0 {
			fmt.Printf("%v", m.Paths)

			y, err := yaml.Marshal(m.Substitutions)
			if err != nil {
				return err
			}
			fmt.Printf("\nAvailable for substitution: \n\n" + string(y))

		}
	}

	return nil
}
