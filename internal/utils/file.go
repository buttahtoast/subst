package utils

import (
	"encoding/json"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type File struct {
	data []byte
	Path string
}

func NewFile(path string) (*File, error) {
	result := &File{
		Path: path,
	}
	data, err := result.load()
	if err != nil {
		return nil, err
	}
	result.data = data
	return result, err
}

func (f *File) load() ([]byte, error) {
	var data []byte
	data, err := ioutil.ReadFile(f.Path)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *File) Byte() []byte {
	return f.data
}

func (f *File) Map() (map[interface{}]interface{}, error) {
	data := make(map[interface{}]interface{})
	err := yaml.Unmarshal(f.data, &data)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *File) YAML() ([]byte, error) {
	d, err := f.Map()
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(d)
}

func (f *File) JSON() ([]byte, error) {
	d, err := f.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(d)
}

func (f *File) SPRUCE() (map[interface{}]interface{}, error) {
	data, err := f.YAML()
	if err != nil {
		return nil, err
	}
	return ParseYAML(data)
}
