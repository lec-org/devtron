package i18n

import (
	"bytes"
	"encoding/json"
)

// MarshalResponse 同时覆盖结果对象中的系统 message；JSON 数字、资源名称和未知数据保持原值。
func MarshalResponse(value interface{}) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil || !bytes.Contains(data, []byte(`"message"`)) {
		return data, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var document interface{}
	if err = decoder.Decode(&document); err != nil {
		return nil, err
	}
	changed := false
	var visit func(interface{})
	visit = func(node interface{}) {
		switch current := node.(type) {
		case map[string]interface{}:
			for key, value := range current {
				if key == "message" {
					if message, ok := value.(string); ok {
						translated := Translate(message)
						if translated != message {
							current[key] = translated
							changed = true
						}
					}
				} else {
					visit(value)
				}
			}
		case []interface{}:
			for _, value := range current {
				visit(value)
			}
		}
	}
	visit(document)
	if !changed {
		return data, nil
	}
	return json.Marshal(document)
}
