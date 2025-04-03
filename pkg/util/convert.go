package util

import (
	"github.com/goccy/go-json"
)

// AnyToStruct converts any to struct, using json marshalling. This is faster than using reflect, e.g. mapstructure
func AnyToStruct[T any](any any) (*T, error) {
	data, err := json.Marshal(any)
	if err != nil {
		return nil, err
	}
	var dst T
	return &dst, json.Unmarshal(data, &dst)
}
