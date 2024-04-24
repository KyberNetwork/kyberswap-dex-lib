package compound

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"
)

func TestMsgpackMarshalUnmarshal(t *testing.T) {
	var pools []*CompoundPool
	{
		precisionA := big.NewInt(1)             // DAI
		precisionB := big.NewInt(1000000000000) // USDC
		// we cannot use the rate from factory as is (it's just exchangeRateStored, without supplyRatePerBlock... like in actual contract)
		// so manually calculate the rates instead
		curBlock := big.NewInt(17484284)
		rateStoredA, _ := new(big.Int).SetString("b839d9be811a1fd7f6ad81", 16)
		supplyRateA, _ := new(big.Int).SetString("1393db059", 16)
		oldBlockA, _ := new(big.Int).SetString("10ac9ba", 16)
		rateStoredB, _ := new(big.Int).SetString("d02a08ebd736", 16)
		supplyRateB, _ := new(big.Int).SetString("2292c55b6", 16)
		oldBlockB, _ := new(big.Int).SetString("010ac9ea", 16)
		storedRateA := new(big.Int).Add(rateStoredA,
			new(big.Int).Div(
				new(big.Int).Mul(new(big.Int).Mul(rateStoredA, supplyRateA), new(big.Int).Sub(curBlock, oldBlockA)),
				bignumber.BONE,
			),
		)
		storedRateB :=
			new(big.Int).Add(rateStoredB,
				new(big.Int).Div(
					new(big.Int).Mul(new(big.Int).Mul(rateStoredB, supplyRateB), new(big.Int).Sub(curBlock, oldBlockB)),
					bignumber.BONE,
				),
			)
		p, err := NewPoolSimulator(entity.Pool{
			Exchange: "",
			Type:     "",
			Reserves: entity.PoolReserves{"6821027635846033", "21272421810258792"},
			Tokens:   []*entity.PoolToken{{Address: "A"}, {Address: "B"}},
			Extra: fmt.Sprintf("{\"swapFee\": \"%v\", \"adminFee\": \"%v\", \"a\": \"%v\", \"rates\": [\"%v\", \"%v\"]}",
				"4000000",
				"5000000000",
				4500,
				storedRateA.String(), storedRateB.String(),
			),
			StaticExtra: fmt.Sprintf("{\"lpToken\": \"LP\", \"precisionMultipliers\": [\"%v\", \"%v\"], \"underlyingTokens\": [\"%v\", \"%v\"]}",
				precisionA.String(), precisionB.String(),
				"Au", "Bu"),
		})
		require.NoError(t, err)
		pools = append(pools, p)
	}
	for _, pool := range pools {
		b, err := pool.MarshalMsg(nil)
		require.NoError(t, err)
		actual := new(CompoundPool)
		_, err = actual.UnmarshalMsg(b)
		require.NoError(t, err)
		require.Empty(t, cmp.Diff(pool, actual, testutil.CmpOpts(CompoundPool{})...))
	}
}
