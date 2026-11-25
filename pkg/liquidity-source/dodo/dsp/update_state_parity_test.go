package dsp

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
	require.NoError(t, json.Unmarshal([]byte(`{"address":"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb","swapFee":3000000000000000,"exchange":"dodo-dsp","type":"dodo-dsp","timestamp":1716870877,"reserves":["233467","1117670600914973"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","name":"Tether USD","symbol":"USDT","decimals":6,"weight":50,"swappable":true},{"address":"0x0c1cf6883efa1b496b01f654e247b9b419873054","name":"SushiSwap LP Token","symbol":"SLP","decimals":18,"weight":50,"swappable":true}],"extra":"{\"i\":\"538000000000000000000000000\",\"K\":\"100000000000000000\",\"B\":\"233467\",\"Q\":\"1117670600914973\",\"B0\":\"1036546\",\"Q0\":\"536995355922673\",\"R\":\"1\",\"mtFeeRate\":\"600000000000000\",\"lpFeeRate\":\"2400000000000000\",\"swappable\":true}","staticExtra":"{\"poolId\":\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\",\"lpToken\":\"0xd55128cbdba933bcf9b5f508108129ffe7e2e9bb\",\"type\":\"DSP\",\"tokens\":[\"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9\",\"0x0c1cf6883efa1b496b01f654e247b9b419873054\"],\"dodoV1SellHelper\":\"0xa5f36e822540efd11fcd77ec46626b916b217c3e\"}"}`), &ent))
	testutil.TestUpdateStateParity(t, func() any {
		sim, err := NewPoolSimulator(ent)
		require.NoError(t, err)
		return sim
	}, []string{"1000", "1000000", "1000000000", "1000000000000000", "1000000000000000000"})
}
