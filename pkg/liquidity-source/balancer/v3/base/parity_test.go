package base_test

// Parity of native CalcAmtOut and legacy CalcAmountOut (the latter lives in legacy_test.go). Remove with the
// legacy methods and the golden data.

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const stablePoolJSON = `{"address":"0xc4ce391d82d164c166df9c8336ddf84206b2f812","exchange":"balancer-v3-stable","type":"balancer-v3-stable","timestamp":1751293016,"reserves":["687804073931103275644","1783969556654743519024"],"tokens":[{"address":"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"},{"address":"0x7f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"}],"extra":"{\"hook\":{},\"fee\":\"20000000000000\",\"aggrFee\":\"500000000000000000\",\"balsE18\":[\"694069210892948295209\",\"2124492373418339554414\"],\"decs\":[\"1\",\"1\"],\"rates\":[\"1009108897721464489\",\"1190879275654308905\"],\"buffs\":[{\"dRate\":[\"976255\",\"976255817341\",\"976255817341645373\",\"976255817341645373456045\",\"976255817341645373456045753577\"],\"rRate\":[\"1024321\",\"1024321681096\",\"1024321681096877127\",\"1024321681096877127977750\",\"1024321681096877127977750950000\"]},{\"dRate\":[\"996629\",\"996629442697\",\"996629442697471179\",\"996629442697471179789157\",\"996629442697471179789157582365\"],\"rRate\":[\"1003381\",\"1003381956380\",\"1003381956380303285\",\"1003381956380303285385258\",\"1003381956380303285385258382000\"]}],\"surge\":{},\"ampParam\":\"5000000\"}","staticExtra":"{\"buffs\":[\"0x0fe906e030a44ef24ca8c7dc7b7c53a6c4f00ce9\",\"0x775f661b0bd1739349b9a2a3ef60be277c5d2d29\"]}","blockNumber":22817774}`

func TestCalcAmtOutParity(t *testing.T) {
	t.Parallel()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(stablePoolJSON), &ep))
	sim, err := stable.NewPoolSimulator(pool.FactoryParams{EntityPool: ep})
	require.NoError(t, err)
	testutil.TestCalcAmtOutParity(t, sim, []string{
		"1", "799999", "842557", "1000000", "1000000000", "123456789012", "2000000000000000000",
		"500000000000000000000", "10000000000000000000000",
	}, nil)
}
