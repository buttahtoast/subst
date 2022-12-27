package tool

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/buttahtoast/subst/pkg/config"
	"github.com/buttahtoast/subst/pkg/utils"
	"github.com/buttahtoast/subst/pkg/wrapper/ejson"
	"github.com/geofffranks/spruce"
	"gopkg.in/yaml.v2"
)

type Spruce struct {
	Subst struct {
		Env     map[string]string           `yaml:"env"`
		Vars    map[interface{}]interface{} `yaml:"vars"`
		Secrets map[interface{}]interface{} `yaml:"secrets"`
	} `yaml:"subst"`
}

func (s *Spruce) ToInterface() (map[interface{}]interface{}, error) {
	tmp := make(map[interface{}]interface{})
	yml, err := yaml.Marshal(s)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(yml, tmp)
	return tmp, err

}

var (
	state = Spruce{}
	paths []string
	err   error
	cfg   config.Configuration
)

func Gather(config config.Configuration) ([]map[interface{}]interface{}, error) {
	cfg = config

	readEnvironmentVariables()

	paths, err = utils.GetKustomizePaths(config.RootDirectory)
	if err != nil {
		return nil, err
	}

	// Append Root Directory to paths
	paths = append(paths, config.RootDirectory)

	// decrypt ejson
	ejs, err := ejson.New(config)
	if err != nil {
		return nil, err
	}

	err = loopPath(VarFiles)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%v", state.Subst.Vars)

	// Read Var Files

	err = loopPath(ejs.Walk)
	if err != nil {
		return nil, err
	}
	state.Subst.Secrets = ejs.GetSecrets()

	manifests, err := Merge(config, state)
	if err != nil {
		return nil, err
	}
	//fmt.Printf("%v", manifests)

	return manifests, err

}

func VarFiles(path string, info fs.FileInfo, err error) error {
	if filepath.Ext(path) == cfg.VarFilePattern {
		file, err := NewFile(path)
		if err != nil {
			return err
		}
		data := file.Map()
		state.Subst.Vars, err = spruce.Merge(state.Subst.Vars, data)
		if err != nil {
			return err
		}
	}
	return err
}

func loopPath(fn filepath.WalkFunc) error {
	for path := range paths {
		err := filepath.Walk(paths[path], fn)
		if err != nil {
			return err
		}
	}
	return nil
}

func readEnvironmentVariables() {
	state.Subst.Env = make(map[string]string)
	r, _ := regexp.Compile("^ARGOCD_ENV_.*$")
	for _, e := range os.Environ() {
		pair := strings.SplitN(e, "=", 2)
		key := pair[0]
		value := pair[1]

		// Rewrite ArgoCD Environment Variables
		if r.MatchString(key) {
			key = strings.ReplaceAll(key, "ARGOCD_ENV_", "")
		}
		state.Subst.Env[key] = value
	}
}
