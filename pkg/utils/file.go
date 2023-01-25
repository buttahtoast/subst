package utils

import (
	"encoding/json"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type file struct {
	data map[interface{}]interface{}
	path string
}

func NewFile(path string) (*file, error) {
	result := &file{
		path: path,
	}
	err := result.loadMap()
	return result, err
}

func (f *file) loadMap() error {
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

func (f *file) load() ([]byte, error) {
	var data []byte
	data, err := ioutil.ReadFile(f.path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *file) Map() map[interface{}]interface{} {
	return f.data
}

func (f *file) YAML() ([]byte, error) {
	return yaml.Marshal(f.data)
}

func (f *file) JSON() ([]byte, error) {
	return json.Marshal(f.data)
}
