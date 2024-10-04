package main

import (
	"path"
)

type YamlPath struct {
	Path string
}

func (y *YamlPath) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	y.Path = path.Clean(s)
	return nil
}

func (y *YamlPath) MarshalYAML() (interface{}, error) {
	return y.Path, nil
}
