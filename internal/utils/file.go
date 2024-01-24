package utils

import (
	"encoding/json"
	"io"
	"os"

	"gopkg.in/yaml.v2"
)

type File struct {
	data []byte
	Path string
}

func NewFile(path string) (*File, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	return &File{data: data, Path: path}, nil
}

func (f *File) OverwriteDataByte(data []byte) {
	f.data = data
}

func (f *File) Byte() []byte {
	return f.data
}

func (f *File) Map() (map[interface{}]interface{}, error) {
	data := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(f.data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func (f *File) YAML() ([]byte, error) {
	return yaml.Marshal(f.data)
}

func (f *File) JSON() ([]byte, error) {
	d, err := f.Map()
	if err != nil {
		return nil, err
	}
	return json.Marshal(d)
}

func (f *File) SPRUCE() (map[interface{}]interface{}, error) {
	return ParseYAML(f.Byte())
}
