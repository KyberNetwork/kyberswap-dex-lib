package bootstrap

import (
	"github.com/goccy/go-json"
)

func PropertiesToStruct(properties map[string]interface{}, o interface{}) error {
	data, err := json.Marshal(properties)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, o)
}
