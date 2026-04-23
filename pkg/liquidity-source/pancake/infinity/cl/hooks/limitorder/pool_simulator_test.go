package limitorder

import (
	"encoding/json"
	"fmt"
	"testing"

	_ "embed"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/cl"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/stretchr/testify/assert"
)

var (
	//go:embed sample_lo_pool.json
	loPoolData string
)

func TestCalcAmountOutWithLoPool(t *testing.T) {
	t.Parallel()

	var poolEnt entity.Pool

	assert.NoError(t, json.Unmarshal([]byte(loPoolData), &poolEnt))

	// only bsc chain
	fmt.Println(len(cl.HookFactories))
	pSim, err := cl.NewPoolSimulator(poolEnt, valueobject.ChainID(56))
	assert.NoError(t, err)

	got, err := pSim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c",
			Amount: bignumber.NewBig10("1000000000000000000"),
		},
		TokenOut: "0x55d398326f99059ff775485246999027b3197955",
	})

	t.Logf("got: %v", got.TokenAmountOut.Amount)
	assert.NoError(t, err)
}
