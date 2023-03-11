package cmd

import (
	"fmt"
	"io"
	"os"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

var (
	cfgFile string
	v       string
)

func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subst",
		Short: "Kustomize with subsitution",
		Long: heredoc.Doc(`
			Create Kustomize builds with stronmg substitution capabilities`),
		SilenceUsage: true,
	}

	//Here is where we define the PreRun func, using the verbose flag value
	//We use the standard output for logs.
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if err := setUpLogs(os.Stdout, v); err != nil {
			return err
		}
		return nil
	}

	//Default value is the warn level
	cmd.PersistentFlags().StringVarP(&v, "verbosity", "v", logrus.WarnLevel.String(), "Log level (debug, info, warn, error, fatal, panic")

	cmd.AddCommand(newDiscoverCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newGenerateDocsCmd())
	cmd.AddCommand(newRenderCmd())
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

// setUpLogs set the log output ans the log level
func setUpLogs(out io.Writer, level string) error {
	logrus.SetOutput(out)
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}
	logrus.SetLevel(lvl)
	return nil
}

func addCommonFlags(flags *flag.FlagSet) {
	flags.StringVar(&cfgFile, "config", "", "Config file")
	flags.String("root-dir", ".", heredoc.Doc(`
			Root directory`))
	flags.String("file-regex", "(subst\\.yaml|.*(ejson|vars))", heredoc.Doc(`
			Regex Pattern to discover substitution files`))
	flags.StringSlice("extra-dirs", []string{}, heredoc.Doc(`
			Additional directories to search for substitution files`))
	flags.Bool("debug", false, heredoc.Doc(`
			Print CLI calls of external tools to stdout (caution: setting this may
			expose sensitive data)`))
}
