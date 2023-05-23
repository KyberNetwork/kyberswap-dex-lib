package source

import (
	"encoding/json"
)

func PropertiesToStruct(properties map[string]interface{}, o interface{}) error {
	data, err := json.Marshal(properties)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, o)
}
