package v2

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// An Ethereum USDC/USDT pool without a hook, at block 26161891. Swapping 10% of a reserve in borrows.
const cloneStatePoolData = `{"address":"0x0bf9244f5f50d79497f58564ee3ab77ef38e28a8","exchange":"uniswap-v4-euler-v2","type":"euler-swap-v2","timestamp":1791633600,"reserves":["150237452340","249779030309"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"er0\":\"200000000000\",\"er1\":\"200000000000\",\"mr0\":\"0\",\"mr1\":\"0\",\"px\":\"1\",\"py\":\"1\",\"cx\":\"999000000000000000\",\"cy\":\"999000000000000000\",\"f0\":\"50000000000000\",\"f1\":\"50000000000000\",\"sh\":\"0x0000000000000000000000000000000000000000\",\"p\":1,\"sv\":[{\"c\":\"197164130437\",\"d\":\"25025946725\",\"mD\":\"72789290051731\",\"tB\":\"2013545817830\",\"eAA\":\"0\",\"bC\":\"67500000000000\",\"dP\":\"999865100000\",\"vP\":[\"999865100000\",\"999054530000\"],\"vVP\":[\"999865100000\",\"999054530000\"],\"ltv\":[0,9400],\"vLtv\":[0,9400],\"iCE\":true},{\"c\":\"80576338998\",\"d\":\"0\",\"mD\":\"49635325580511\",\"tB\":\"284098080490\",\"eAA\":\"38930898295\",\"bC\":\"45000000000000\",\"dP\":\"999054530000\",\"vP\":[\"999865100000\",\"999054530000\"],\"vVP\":[\"999865100000\",\"999054530000\"],\"ltv\":[9400,0],\"vLtv\":[9400,0]}],\"bv\":[{\"c\":\"197164130437\",\"d\":\"25025946725\",\"mD\":\"72789290051731\",\"tB\":\"2013545817830\",\"eAA\":\"0\",\"bC\":\"67500000000000\",\"dP\":\"999865100000\",\"vP\":[\"999865100000\",\"999054530000\"],\"vVP\":[\"999865100000\",\"999054530000\"],\"ltv\":[0,9400],\"vLtv\":[0,9400],\"iCE\":true},{\"c\":\"80576338998\",\"d\":\"0\",\"mD\":\"49635325580511\",\"tB\":\"284098080490\",\"eAA\":\"38930898295\",\"bC\":\"45000000000000\",\"dP\":\"999054530000\",\"vP\":[\"999865100000\",\"999054530000\"],\"vVP\":[\"999865100000\",\"999054530000\"],\"ltv\":[9400,0],\"vLtv\":[9400,0]},{\"c\":\"197164130437\",\"d\":\"25025946725\",\"mD\":\"72789290051731\",\"tB\":\"2013545817830\",\"eAA\":\"0\",\"bC\":\"67500000000000\",\"dP\":\"999865100000\",\"vP\":[\"999865100000\",\"999054530000\"],\"vVP\":[\"999865100000\",\"999054530000\"],\"ltv\":[0,9400],\"vLtv\":[0,9400],\"iCE\":true}],\"cV\":\"0x797dd80692c3b2dadabce8e30c07fde5307d48a9\",\"c\":[\"0\",\"38930898295\"]}","staticExtra":"{\"sv0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"sv1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"bv0\":\"0x797DD80692c3b2dAdabCe8e30C07fDE5307D48a9\",\"bv1\":\"0x313603FA690301b0CaeEf8069c065862f9162162\",\"ea\":\"0x0e0c281ff05D34729Cd764DcfC4Fa999b720407c\",\"evc\":\"0x0C9a3dd6b8F28529d72d7f9cE918D493519EE383\"}","blockNumber":26161891}`

// Routers keep one simulator per pool and give each request a CloneState copy to swap on. A swap
// applied to the clone must leave the original exactly as built, or every later quote on that
// pool sees liquidity that does not exist.
func TestUpdateBalanceOnCloneLeavesOriginalUntouched(t *testing.T) {
	t.Parallel()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(cloneStatePoolData), &ep))
	original, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	asBuilt, err := NewPoolSimulator(ep)
	require.NoError(t, err)

	for i := range 2 {
		clone := original.CloneState()
		tokenIn := pool.TokenAmount{Token: ep.Tokens[i].Address,
			Amount: new(big.Int).Div(original.GetReserves()[i], big.NewInt(10))}
		res, err := pool.CalcAmountOut(context.Background(), clone, tokenIn, ep.Tokens[1-i].Address, nil)
		require.NoError(t, err)
		clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: tokenIn,
			TokenAmountOut: *res.TokenAmountOut, SwapInfo: res.SwapInfo})
		require.Equal(t, asBuilt, original, "swap %d on the clone changed the original", i)
	}
}
