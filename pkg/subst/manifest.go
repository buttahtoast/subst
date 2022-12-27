package subst

import (
	"encoding/json"
	"fmt"
	"log"

	"go.opencensus.io/resource"

	"gopkg.in/yaml.v2"
)

type manifest struct {
	Data       map[interface{}]interface{}
	Eval       bool
	Substitute bool
	Decrypt    bool
}

func NewManifest(data resource.Resource) (manifest, error) {
	var res map[interface{}]interface{}
	result := manifest{}

	// Unmarshal Data
	d, _ := data.AsYAML()
	err := yaml.Unmarshal(d, &res)
	if err != nil {
		return result, err
	}

	result := manifest{
		data: data,
	}
	err := result.loadMap()
	return result, err
}

func (m *manifest) load() error {

	m, _ := m.data.AsYAML()
	var tmp map[interface{}]interface{}
	log.Printf("HERE 1")
	err := yaml.Unmarshal(m, &tmp)
	if err != nil {
		return nil, err
	}

	data, err := f.load()
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(data, &f.data)
	if err != nil {
		return err
	}

	if f.data == nil {
		f.data = make(map[interface{}]interface{})
	}

	return nil
}

func (m *manifest) Map() map[interface{}]interface{} {
	return m.data
}

/* create a golang function which prints map[interface{}]interface{} as yaml */
func (m *manifest) PrintYAML() {
	y, err := yaml.Marshal(m.data)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("---\n%s\n", string(y))
}

/* create a golang function which prints map[interface{}]interface{} as json */
func (m *manifest) PrintJSON() {
	j, err := json.Marshal(m.data)
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Printf("%s\n", string(j))
}
