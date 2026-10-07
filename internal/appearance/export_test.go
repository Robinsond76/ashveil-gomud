package appearance

import "gopkg.in/yaml.v2"

func yamlUnmarshal(raw []byte, v any) error { return yaml.Unmarshal(raw, v) }
