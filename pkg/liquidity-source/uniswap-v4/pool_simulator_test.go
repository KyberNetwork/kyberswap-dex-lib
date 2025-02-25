package uniswapv4

import (
	_ "embed"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	utils "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	//go:embed sample_pool.json
	poolData string
)

func TestPoolSimulator(t *testing.T) {
	var (
		chainID = 1
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	t.Log(poolEnt)
	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2",
			Amount: utils.NewBig10("1000000000000000000"),
		},
		TokenOut: "0xbeab712832112bd7664226db7cd025b153d3af55",
	})
	assert.NoError(t, err)
	assert.Equal(t, utils.NewBig10("2376445698940"), got.TokenAmountOut.Amount)
}
