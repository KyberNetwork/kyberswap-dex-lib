package saddle

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// MsgpackTestPools ...
func MsgpackTestPools() []*PoolSimulator {
	var pools []*PoolSimulator
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907", "553429429583268691085"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"48000\",\"futureA\":\"92000\",\"initialATime\":1652287436,\"futureATime\":1653655053,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\"]}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"20288190723295606376", "9812867150429539713", "29980929628444248071"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra:       "{\"initialA\":\"10000\",\"futureA\":\"20000\",\"initialATime\":1620946481,\"futureATime\":1622245581,\"swapFee\":\"8000000\",\"adminFee\":\"9999999999\",\"defaultWithdrawFee\":\"0\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\"], \"totalSupply\": \"29980929628444248071\"}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"339028421564024338437", "347684462442560871352", "423798212946198474118", "315249216225911580289", "1404290718401538825321"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}, {Address: "D"}},
			Extra:       "{\"initialA\":\"60000\",\"futureA\":\"60000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"1000000\",\"adminFee\":\"10000000000\",\"defaultWithdrawFee\":\"5000000\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\",\"1\"]}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"233518765839", "198509040315", "228986742536043517345011", "654251953025609178732174"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"18000\",\"futureA\":\"120000\",\"initialATime\":1627094541,\"futureATime\":1627699238,\"swapFee\":\"2000000\",\"adminFee\":\"10000000000\", \"defaultWithdrawFee\":\"5000000\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\",\"1\"]}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	{
		p, err := NewPoolSimulator(entity.Pool{
			Exchange:    "",
			Type:        "",
			Reserves:    entity.PoolReserves{"64752405287155128155", "426593278742302082683", "66589357932477536907", "553429429583268691085"},
			Tokens:      []*entity.PoolToken{{Address: "A"}, {Address: "B"}, {Address: "C"}},
			Extra:       "{\"initialA\":\"48000\",\"futureA\":\"92000\",\"initialATime\":1652287436,\"futureATime\":1653655053,\"swapFee\":\"4000000\",\"adminFee\":\"5000000000\"}",
			StaticExtra: "{\"lpToken\":\"LP\",\"precisionMultipliers\":[\"1\",\"1\",\"1\"]}",
		})
		if err != nil {
			panic(err)
		}
		pools = append(pools, p)
	}
	return pools
}
