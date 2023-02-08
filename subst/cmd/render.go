package cmd

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/subst"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
)

func newRenderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "render",
		Short: "Render structure with substitutions",
		Long: heredoc.Doc(`
			Run 'subst discover' to return directories that contain plugin compatible files. Mainly used for automatic plugin discovery by ArgoCD`),
		RunE: render,
	}

	flags := cmd.Flags()
	addCommonFlags(flags)
	addRenderFlags(flags)
	return cmd

}

func addRenderFlags(flags *flag.FlagSet) {
	if flags.Lookup("kubeconfig") == nil {
		flags.String("kubeconfig", "", "Path to a kubeconfig")
	}
	flags.String("ejson-secret", "", heredoc.Doc(`
	        Specify EJSON Secret name (each key within the secret will be used as a decryption key)`))
	flags.String("ejson-namespace", "", heredoc.Doc(`
	        Specify EJSON Secret namespace`))
	flags.String("env-regex", "^ARGOCD_ENV_.*$", heredoc.Doc(`
	        Only expose environment variables that match the given regex`))
	flags.StringSlice("ejson-key", []string{}, heredoc.Doc(`
			Specify EJSON Private key used for decryption.
			May be specified multiple times or separate values with commas`))
	flags.Bool("must-decrypt", false, heredoc.Doc(`
			Fail if not all ejson files can be decrypted`))
	flags.Bool("skip-decrypt", false, heredoc.Doc(`
			Disable decryption of EJSON files`))
	flags.Bool("skip-eval", false, heredoc.Doc(`
			Skip Spruce evaluation for all files (Useful if required variables are not available)`))

}

func render(cmd *cobra.Command, args []string) error {
	configuration, err := config.LoadConfiguration(cfgFile, cmd)
	if err != nil {
		return fmt.Errorf("failed loading configuration: %w", err)
	}
	m, err := subst.New(*configuration)
	if err != nil {
		return err
	}
	if m != nil {
		err = m.Build()
		if err != nil {
			return err
		}
		if m.Manifests != nil {
			for _, f := range m.Manifests {
				utils.PrintYAML(f)
			}
		}
	}

	return nil
}
