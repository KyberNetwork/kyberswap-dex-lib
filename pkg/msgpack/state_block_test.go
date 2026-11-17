package msgpack

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/integral"
	algebrav1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/machima"
	nativev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v3"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	sbToken0 = "0x2222222222222222222222222222222222222222"
	sbToken1 = "0x3333333333333333333333333333333333333333"
)

// A simulator is encoded as a positional array, so a field added to the encoding would break a
// snapshot that another version wrote or reads. The StateBlock is not part of the encoding: the same
// pool with and without sB in its Extra encodes to the same simulator, and a decoded simulator has no
// StateBlock, so its meta falls back to blockNumber. A pancake-infinity-cl simulator is not in the
// table: its hook does not decode from msgpack, with or without the StateBlock (it panics on main too).
func TestStateBlockStaysOutOfMsgpack(t *testing.T) {
	const ticks = `"ticks":[{"index":-60,"liquidityGross":1000,"liquidityNet":1000},{"index":60,"liquidityGross":1000,"liquidityNet":-1000}]`
	v3Extra := func(sB string) string {
		return `{"liquidity":1000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` + ticks + sB + `}`
	}
	vault := `{"DepositPaused":false,"RedeemPaused":false,"MinDeposit":"0","ExchangeRate":"0","MinRedeemInterval":"0","RedeemCoolDownExempt":false}`
	machimaStatic, err := json.Marshal(machima.StaticExtra{Token: sbToken1, RouterAddress: sbToken0, WETH: sbToken0,
		USDC: "0x4444444444444444444444444444444444444444", XMA: "0x5555555555555555555555555555555555555555"})
	require.NoError(t, err)

	for name, tc := range map[string]struct {
		extra       func(sB string) string // the Extra with the sB field, or without it for ""
		staticExtra string
		newSim      func(entity.Pool) (pool.IPoolSimulator, error)
	}{
		"uniswap-v3": {v3Extra, "", func(p entity.Pool) (pool.IPoolSimulator, error) {
			return uniswapv3.NewPoolSimulator(p, valueobject.ChainIDEthereum)
		}},
		"algebra-v1": {func(sB string) string {
			return `{"liquidity":1000,"globalState":{"price":79228162514264337593543950336,"tick":0,"feeZto":100,"feeOtz":100,` +
				`"timepoint_index":0,"community_fee_token0":0,"community_fee_token1":0,"unlocked":true},` + ticks + `,"tickSpacing":60` + sB + `}`
		}, "", func(p entity.Pool) (pool.IPoolSimulator, error) { return algebrav1.NewPoolSimulator(p) }},
		"algebra-integral": {func(sB string) string {
			return `{"liq":"1000","gS":{"price":"79228162514264337593543950336","tick":0,"lF":100,"pC":0,"cF":0,"un":true},` +
				`"ticks":[{"Index":-60,"LiquidityGross":"1000","LiquidityNet":"1000"},{"Index":60,"LiquidityGross":"1000","LiquidityNet":"-1000"}],"tS":60` + sB + `}`
		}, `{}`, func(p entity.Pool) (pool.IPoolSimulator, error) { return integral.NewPoolSimulator(p) }},
		"native-v3": {func(sB string) string {
			return `{"unlocked":true,"liquidity":1000,"sqrtPriceX96":79228162514264337593543950336,"tick":0,` + ticks +
				`,"vaults":[` + vault + `,` + vault + `]` + sB + `}`
		}, `{"tickSpacing":60,"scan":true}`, func(p entity.Pool) (pool.IPoolSimulator, error) {
			return nativev3.NewPoolSimulator(p, valueobject.ChainIDEthereum)
		}},
		"machima": {v3Extra, string(machimaStatic), func(p entity.Pool) (pool.IPoolSimulator, error) {
			return machima.NewPoolSimulator(p, valueobject.ChainIDBase)
		}},
	} {
		t.Run(name, func(t *testing.T) {
			newPool := func(sB string) entity.Pool {
				return entity.Pool{
					Address: "0x1111111111111111111111111111111111111111", Type: name, Exchange: name, SwapFee: 3000,
					BlockNumber: 90,
					Reserves:    entity.PoolReserves{"1000000000000000000", "1000000000000000000"},
					Tokens: []*entity.PoolToken{{Address: sbToken0, Decimals: 18, Swappable: true},
						{Address: sbToken1, Decimals: 18, Swappable: true}},
					Extra: tc.extra(sB), StaticExtra: tc.staticExtra,
				}
			}
			roundTrip := func(p entity.Pool) (with pool.IPoolSimulator, decoded pool.IPoolSimulator) {
				sim, err := tc.newSim(p)
				require.NoError(t, err)
				encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{p.Address: sim})
				require.NoError(t, err)
				out, err := DecodePoolSimulatorsMap(encoded)
				require.NoError(t, err)
				return sim, out[p.Address]
			}

			stamped, decodedStamped := roundTrip(newPool(`,"sB":95`))
			_, decodedPlain := roundTrip(newPool(``))

			// the map encoding is not deterministic, so compare what decodes: an encoded StateBlock would differ
			assert.Empty(t, cmp.Diff(decodedPlain, decodedStamped, testutil.CmpOpts()...),
				"the StateBlock must not be part of the msgpack encoding")

			meta := func(sim pool.IPoolSimulator) string {
				out, err := json.Marshal(sim.GetMetaInfo(sbToken0, sbToken1))
				require.NoError(t, err)
				return string(out)
			}
			assert.Contains(t, meta(stamped), `"sB":95`, "the simulator built from the Extra exposes sB")
			assert.Contains(t, meta(stamped), `"blockNumber":90`)
			assert.NotContains(t, meta(decodedStamped), "sB", "a decoded simulator falls back to blockNumber")
			assert.Contains(t, meta(decodedStamped), `"blockNumber":90`)
		})
	}
}
