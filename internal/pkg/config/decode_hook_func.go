package config

import (
	"math/big"
	"reflect"

	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog/log"
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
			log.Err(err).Msg("StringToBigIntHookFunc Error when parse float")
			return nil, nil
		}
		res, _ := value.Int(new(big.Int))

		return res, nil
	}
}
