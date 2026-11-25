package dvm

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
	require.NoError(t, json.Unmarshal([]byte(`{"address":"0x68276dc302d390245f3382eb4d2ea3a9317d46ef","swapFee":3000000000000000,"exchange":"dodo-dvm","type":"dodo-dvm","timestamp":1716863956,"reserves":["15580539464573","54845488636364795"],"tokens":[{"address":"0x82af49447d8a07e3bd95bd0d56f35241523fbab1","name":"Wrapped Ether","symbol":"WETH","decimals":18,"weight":50,"swappable":true},{"address":"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1","name":"Dai Stablecoin","symbol":"DAI","decimals":18,"weight":50,"swappable":true}],"extra":"{\"i\":\"100000\",\"K\":\"1000000000000000000\",\"B\":\"15580539464573\",\"Q\":\"54845488636364795\",\"B0\":\"2923221347601320894515\",\"Q0\":\"0\",\"R\":\"1\",\"mtFeeRate\":\"600000000000000\",\"lpFeeRate\":\"2400000000000000\",\"swappable\":true}","staticExtra":"{\"poolId\":\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\",\"lpToken\":\"0x68276dc302d390245f3382eb4d2ea3a9317d46ef\",\"type\":\"DVM\",\"tokens\":[\"0x82af49447d8a07e3bd95bd0d56f35241523fbab1\",\"0xda10009cbd5d07dd0cecc66161fc93d7c9000da1\"],\"dodoV1SellHelper\":\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\"}"}`), &ent))
	testutil.TestUpdateStateParity(t, func() any {
		sim, err := NewPoolSimulator(ent)
		require.NoError(t, err)
		return sim
	}, []string{"1000", "1000000", "1000000000", "1000000000000000", "1000000000000000000"})
}
