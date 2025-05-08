package lb

import (
	_ "embed"
	"testing"

	"github.com/goccy/go-json"
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

func TestCalcAmountOut(t *testing.T) {
	var (
		chainID = 1
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
			Amount: utils.NewBig10("1000000000000000000"),
		},
		TokenOut: "0x55d398326f99059ff775485246999027b3197955",
	})
	assert.NoError(t, err)
	assert.Equal(t, utils.NewBig10("999901000000000000"), got.TokenAmountOut.Amount)
}

func TestCalcAmountIn(t *testing.T) {
	var (
		chainID = 1
		poolEnt entity.Pool
	)
	assert.NoError(t, json.Unmarshal([]byte(poolData), &poolEnt))

	pSim, err := NewPoolSimulator(poolEnt, valueobject.ChainID(chainID))
	assert.NoError(t, err)

	got, err := pSim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{
			Token:  "0x55d398326f99059ff775485246999027b3197955",
			Amount: utils.NewBig10("999901000000000000"),
		},
		TokenIn: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
	})
	assert.NoError(t, err)
	assert.Equal(t, utils.NewBig10("1000000000000000000"), got.TokenAmountIn.Amount)
}
