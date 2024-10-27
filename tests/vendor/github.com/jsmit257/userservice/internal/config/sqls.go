package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Sqls map[string]map[string]string

func NewSqls(vendor string) (Sqls, error) {

	f, err := os.Open(fmt.Sprintf("/sql/%s/runtime.yaml", vendor))
	if err != nil {
		return nil, err
	}

	result := make(Sqls, 3)
	if err := yaml.NewDecoder(f).Decode(result); err != nil {
		return nil, err
	}

	return result, nil
}
