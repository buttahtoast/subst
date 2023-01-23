package cmd

import (
	"fmt"
	"os"

	flag "github.com/spf13/pflag"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subst",
		Short: "Kustomize with subsitution",
		Long: heredoc.Doc(`
			Create Kustomize builds with stronmg substitution capabilities`),
		SilenceUsage: true,
	}

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateDocsCmd())
	cmd.AddCommand(newRenderCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newSubstitutionsCmd())
	//

	cmd.DisableAutoGenTag = true

	return cmd
}

// Execute runs the application
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func addCommonFlags(flags *flag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.String("root-dir", ".", heredoc.Doc(`
			Root directory`))
	flags.String("ejson-pattern", ".ejson", heredoc.Doc(`
			Pattern to discover ejson files`))
	flags.String("vars-pattern", ".vars", heredoc.Doc(`
			Pattern to discover var files`))
	flags.StringSlice("extra-dirs", []string{}, heredoc.Doc(`
			Additional directories to search for substitution files`))
	flags.Bool("debug", false, heredoc.Doc(`
			Print CLI calls of external tools to stdout (caution: setting this may
			expose sensitive data)`))
}
