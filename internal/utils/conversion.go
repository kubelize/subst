package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// UnmarshalJSONorYAMLToInterface unmarshals data to interface map
func UnmarshalJSONorYAMLToInterface(data []byte, result *map[interface{}]interface{}) error {
	var tmp map[string]interface{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		err = yaml.Unmarshal(data, &tmp)
		if err != nil {
			return err
		}
	}
	*result = ToInterface(tmp)
	return nil
}

// ToInterface converts a map[string]interface{} to a map[interface{}]interface{}.
func ToInterface(inputMap map[string]interface{}) map[interface{}]interface{} {
	var convertedMap = make(map[interface{}]interface{}, len(inputMap))
	for key, value := range inputMap {
		convertedMap[key] = value
	}
	return convertedMap
}

// PrintJSON prints a map as JSON
func PrintJSON(data map[interface{}]interface{}) error {
	// Convert to map[string]interface{} for JSON marshaling
	stringMap := mapify(data)
	j, err := json.MarshalIndent(stringMap, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(j))
	return nil
}

// PrintYAMLBytes prints YAML from byte slice
func PrintYAMLBytes(data []byte) error {
	writer := bufio.NewWriter(os.Stdout)
	defer writer.Flush()

	if _, err := writer.WriteString("---\n"); err != nil {
		return err
	}
	if _, err := writer.Write(data); err != nil {
		return err
	}

	return nil
}

// mapify converts map[interface{}]interface{} to map[string]interface{}
func mapify(input map[interface{}]interface{}) map[string]interface{} {
	output := make(map[string]interface{}, len(input))
	for k, v := range input {
		switch vv := v.(type) {
		case map[interface{}]interface{}:
			output[k.(string)] = mapify(vv)
		default:
			output[k.(string)] = vv
		}
	}
	return output
}

// DeepMerge recursively merges src into dst
// If a key exists in both maps and both values are maps, it recursively merges them
// Otherwise, src values override dst values
func DeepMerge(dst, src map[string]interface{}) map[string]interface{} {
	for key, srcVal := range src {
		if dstVal, exists := dst[key]; exists {
			// Both dst and src have this key
			srcMap, srcIsMap := srcVal.(map[string]interface{})
			dstMap, dstIsMap := dstVal.(map[string]interface{})
			
			if srcIsMap && dstIsMap {
				// Both are maps, merge recursively
				dst[key] = DeepMerge(dstMap, srcMap)
			} else {
				// Not both maps, src overrides dst
				dst[key] = srcVal
			}
		} else {
			// Key only in src, add it
			dst[key] = srcVal
		}
	}
	return dst
}
