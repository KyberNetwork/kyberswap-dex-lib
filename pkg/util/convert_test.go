package util

import (
	"net/http"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/goccy/go-json"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
)

type AlgebraConfig struct {
	DexID              string
	SubgraphAPI        string      `json:"subgraphAPI"`
	SubgraphHeaders    http.Header `json:"subgraphHeaders"`
	AllowSubgraphError bool        `json:"allowSubgraphError"`
	SkipFeeCalculating bool        `json:"skipFeeCalculating"` // do not pre-calculate fee at tracker, use last block's fee instead
	UseDirectionalFee  bool        `json:"useDirectionalFee"`  // for Camelot and similar dexes

	AlwaysUseTickLens bool
	TickLensAddress   string
}

// BenchmarkAnyToStruct/json
// BenchmarkAnyToStruct/json-16         	  618277	      2227 ns/op
// BenchmarkAnyToStruct/mapstructure
// BenchmarkAnyToStruct/mapstructure-16 	  175678	      6521 ns/op
// BenchmarkAnyToStruct/msgpack
// BenchmarkAnyToStruct/msgpack-16      	  391466	      2592 ns/op
func BenchmarkAnyToStruct(b *testing.B) {
	properties := map[string]any{
		"SubgraphAPI":        "https://api.thegraph.com/subgraphs/name/uniswap/uniswap-v3",
		"SubgraphHeaders":    http.Header{"X-TheGraph-API-Key": []string{"1234567890abcdef"}},
		"AllowSubgraphError": true,
		"SkipFeeCalculating": true,
		"UseDirectionalFee":  true,
		"AlwaysUseTickLens":  true,
		"TickLensAddress":    "0x123456",
	}

	var struct1, struct2 AlgebraConfig
	b.Run("json", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := json.Marshal(properties)
			if err != nil {
				b.Fatal(err)
			}

			err = json.Unmarshal(data, &struct1)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	b.Run("mapstructure", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			err := mapstructure.Decode(properties, &struct2)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	assert.Equal(b, struct1, struct2)
	var struct3 AlgebraConfig
	b.Run("msgpack", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			data, err := msgpack.Marshal(properties)
			if err != nil {
				b.Fatal(err)
			}

			err = msgpack.Unmarshal(data, &struct3)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	assert.Equal(b, struct1, struct3)
}
