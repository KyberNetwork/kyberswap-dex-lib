package pool

import (
	"github.com/goccy/go-json"
)

func PropertiesToStruct(properties map[string]any, outStruct any) error {
	data, err := json.Marshal(properties)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, outStruct)
}
