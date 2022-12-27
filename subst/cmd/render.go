package cmd

import (
	"fmt"
	"log"

	"github.com/MakeNowJust/heredoc"
	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/tool"
	"github.com/spf13/cobra"
	flag "github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
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
	flags.String("ejson-secret", "", heredoc.Doc(`
	        Specify EJSON Secret name (each key within the secret will be used as a decryption key)`))
	flags.String("ejson-namespace", "", heredoc.Doc(`
	        Specify EJSON Secret namespace`))
	flags.StringSlice("ejson-key", []string{}, heredoc.Doc(`
			Specify EJSON Private key used for decryption.
			May be specified multiple times or separate values with commas`))
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
	m, err := tool.Gather(*configuration)
	if err != nil {
		fmt.Println(err)
	}

	for _, f := range m {
		y, err := yaml.Marshal(f)
		if err != nil {
			log.Fatalf("error: %v", err)
		}
		fmt.Printf("---\n%s\n", string(y))
	}
	//return yaml.Marshal(f.data)

	return nil

}
