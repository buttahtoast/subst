package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Printf Configuration",
		Long: heredoc.Doc(`
			Run 'subst config' prints the given configuration (based on configuration files and env)`),
		RunE: configurationcmd,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd

}

func configurationcmd(cmd *cobra.Command, args []string) error {
	configuration, err := config.LoadConfiguration(cfgFile, cmd)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	config.PrintConfiguration(configuration)
	return nil
}
