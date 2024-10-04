package main

import (
	"net/url"
)

type YamlURL struct {
	*url.URL
}

func (j *YamlURL) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err != nil {
		return err
	}
	u, err := url.Parse(s)
	j.URL = u
	return err
}

func (j *YamlURL) MarshalYAML() (interface{}, error) {
	return j.String(), nil
}
