package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type Configuration struct {
	EnvRegex        string        `mapstructure:"env-regex"`
	EnvSubstEnable  bool          `mapstructure:"envsubst"`
	RootDirectory   string        `mapstructure:"root-dir"`
	FileRegex       string        `mapstructure:"file-regex"`
	SecretName      string        `mapstructure:"secret-name"`
	SecretNamespace string        `mapstructure:"secret-namespace"`
	EjsonKey        []string      `mapstructure:"ejson-key"`
	SkipDecrypt     bool          `mapstructure:"skip-decrypt"`
	MustDecrypt     bool          `mapstructure:"must-decrypt"`
	SkipEvaluation  bool          `mapstructure:"skip-eval"`
	KubectlTimeout  time.Duration `mapstructure:"kubectl-timeout"`
	Kubeconfig      string        `mapstructure:"kubeconfig"`
	KubeAPI         string        `mapstructure:"kube-api"`
}

var (
	configLocations = []string{
		".",
	}
)

func LoadConfiguration(cfgFile string, cmd *cobra.Command) (*Configuration, error) {
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
	} else {
		v.SetConfigName("subst")
		if cfgFile, ok := os.LookupEnv("SUBST_CONFIG_DIR"); ok {
			v.AddConfigPath(cfgFile)
		} else {
			for _, searchLocation := range configLocations {
				v.AddConfigPath(searchLocation)
			}
		}
	}

	logrus.Debugf("Using configuration file: %s", v.ConfigFileUsed())

	if err := v.ReadInConfig(); err != nil {
		if cfgFile != "" {
			// Only error out for specified config file. Ignore for default locations.
			return nil, fmt.Errorf("failed loading config file: %w", err)
		}
	}

	cfg := &Configuration{}
	if err := v.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("failed unmarshaling configuration: %w", err)
	}

	// Resolve Root Directory
	rootAbs, err := filepath.Abs(cfg.RootDirectory)
	if err != nil {
		return nil, fmt.Errorf("failed resolving root directory: %w", err)
	} else {
		cfg.RootDirectory = rootAbs
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
