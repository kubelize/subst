package decryptors

import (
	"encoding/json"

	"gopkg.in/yaml.v3"
)

func UnmarshalJSONorYAML(data []byte) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal(data, &result)
	if err != nil {
		err = yaml.Unmarshal(data, &result)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
