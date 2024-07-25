package config

import (
	"log"
	"math/big"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func StringToBigIntHookFunc() mapstructure.DecodeHookFunc {
	return func(
		f reflect.Type,
		t reflect.Type,
		data interface{}) (interface{}, error) {
		if f.Kind() != reflect.String || t != reflect.TypeOf(big.NewInt(0)) {
			return data, nil
		}

		// Return the parsed value
		value, _, err := big.ParseFloat(data.(string), 10, 0, big.ToNearestEven)
		if err != nil {
			log.Printf("StringToBigIntHookFunc Error when parse float %v", err)
			return nil, nil
		}
		res, _ := value.Int(new(big.Int))

		return res, nil
	}
}
