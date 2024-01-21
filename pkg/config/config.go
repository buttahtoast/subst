package config

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	EnvRegex        string        `mapstructure:"env-regex"`
	RootDirectory   string        `mapstructure:"root-dir"`
	FileRegex       string        `mapstructure:"file-regex"`
	SecretName      string        `mapstructure:"secret-name"`
	SecretNamespace string        `mapstructure:"secret-namespace"`
	EjsonKey        []string      `mapstructure:"ejson-key"`
	SkipDecrypt     bool          `mapstructure:"skip-decrypt"`
	MustDecrypt     bool          `mapstructure:"must-decrypt"`
	KubectlTimeout  time.Duration `mapstructure:"kubectl-timeout"`
	Kubeconfig      string        `mapstructure:"kubeconfig"`
	KubeAPI         string        `mapstructure:"kube-api"`
	Output          string        `mapstructure:"output"`
}

func LoadConfiguration(cfgFile string, cmd *cobra.Command, directory string) (*Configuration, error) {
	v := viper.New()

	cmd.Flags().VisitAll(func(flag *flag.Flag) {
		flagName := flag.Name
		if flagName != "config" && flagName != "help" {
			if err := v.BindPFlag(flagName, flag); err != nil {
				panic(fmt.Sprintf("failed binding flag %q: %v\n", flagName, err.Error()))
			}
		}
	})

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	}

	//else {
	//	v.AddConfigPath(directory)
	//	v.SetConfigFile(".subst.yaml")
	//}

	if v.ConfigFileUsed() != "" {
		logrus.Debugf("Using configuration file: %s", v.ConfigFileUsed())
	}

	if err := v.ReadInConfig(); err != nil {
		switch err.(type) {
		case viper.ConfigFileNotFoundError:
			logrus.Debugf("No Config file found, loaded config from Environment")
		default:
			logrus.Fatalf("Error when Fetching Configuration - %s", err)
		}
	}

	cfg := &Configuration{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed unmarshaling configuration: %w", err)
	}

	// Root Directory
	cfg.RootDirectory = directory

	if cfg.SecretName == "" {
		cfg.SecretName = os.Getenv("ARGOCD_APP_NAME")
	}

	if cfg.SecretName != "" {
		regex := regexp.MustCompile(`[^a-zA-Z0-9]+`)
		cfg.SecretName = regex.ReplaceAllString(cfg.SecretName, "-")
	}

	if cfg.SecretName != "" && cfg.SecretNamespace == "" {
		return nil, fmt.Errorf("secret-namespace must be set when --secret-name is set")
	}

	logrus.Debugf("Configuration: %+v\n", cfg)
	return cfg, nil

}

func PrintConfiguration(cfg *Configuration) {
	fmt.Fprintln(os.Stderr, " Configuration")
	e := reflect.ValueOf(cfg).Elem()
	typeOfCfg := e.Type()

	for i := 0; i < e.NumField(); i++ {
		var pattern string
		switch e.Field(i).Kind() {
		case reflect.Bool:
			pattern = "%s: %t\n"
		default:
			pattern = "%s: %s\n"
		}
		fmt.Fprintf(os.Stderr, pattern, typeOfCfg.Field(i).Name, e.Field(i).Interface())
	}
}
