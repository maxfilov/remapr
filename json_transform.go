package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/itchyny/gojq"
)

type Identifiers = map[string]int

type JsonTransform = func([]byte, Identifiers) ([]byte, error)

func NewJQJsonTransform(jq string) (JsonTransform, error) {
	parsedQuery, err := gojq.Parse(jq)
	if err != nil {
		return nil, err
	}
	compiledQuery, err := gojq.Compile(parsedQuery, gojq.WithVariables([]string{"$ids"}))
	if err != nil {
		return nil, err
	}
	return func(input []byte, ids Identifiers) ([]byte, error) {
		var parsed any
		err = json.Unmarshal(input, &parsed)
		if err != nil {
			return nil, err
		}
		run := compiledQuery.Run(parsed, ids)
		var result []byte
		for {
			v, ok := run.Next()
			if !ok {
				break
			}
			if len(result) != 0 {
				return nil, fmt.Errorf("the jq script produced too much data")
			}
			if err, ok = v.(error); ok {
				var haltError *gojq.HaltError
				if errors.As(err, &haltError) && haltError.Value() == nil {
					break
				}
				return nil, fmt.Errorf("jq run error: %w", err)
			}
			result, err = json.Marshal(v)
			if err != nil {
				return nil, err
			}
		}
		return result, nil
	}, nil
}
