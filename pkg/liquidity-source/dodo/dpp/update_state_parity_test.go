package dpp

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// The adapter path (CalcAmtOut, UpdateState without a fee) must reach the same state as UpdateBalance with the fee.
func TestUpdateStateParity(t *testing.T) {
	var ent entity.Pool
	require.NoError(t, json.Unmarshal([]byte(`{"address":"0xb7392c0d85676de049121771c1edb31edd446336","swapFee":500000000000000,"exchange":"dodo-dpp","type":"dodo-dpp","timestamp":1716868655,"reserves":["900000000000000000","100000"],"tokens":[{"address":"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a","name":"Magic Internet Money","symbol":"MIM","decimals":18,"weight":50,"swappable":true},{"address":"0xaf88d065e77c8cc2239327c5edb3a432268e5831","name":"USD Coin","symbol":"USDC","decimals":6,"weight":50,"swappable":true}],"extra":"{\"i\":\"1000000\",\"K\":\"250000000000000\",\"B\":\"900000000000000000\",\"Q\":\"100000\",\"B0\":\"900000000000000000\",\"Q0\":\"100000\",\"R\":\"0\",\"mtFeeRate\":\"0\",\"lpFeeRate\":\"500000000000000\",\"swappable\":true}","staticExtra":"{\"poolId\":\"0xb7392c0d85676de049121771c1edb31edd446336\",\"lpToken\":\"\",\"type\":\"DPP\",\"tokens\":[\"0xfea7a6a0b346362bf88a9e4a88416b77a57d6c2a\",\"0xaf88d065e77c8cc2239327c5edb3a432268e5831\"],\"dodoV1SellHelper\":\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\"}"}`), &ent))
	testutil.TestUpdateStateParity(t, func() any {
		sim, err := NewPoolSimulator(ent)
		require.NoError(t, err)
		return sim
	}, []string{"1000", "1000000", "1000000000", "1000000000000000", "1000000000000000000"})
}
