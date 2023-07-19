package composablestable

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestSwap(t *testing.T) {

	//"{\"address\":\"0x02d928e68d8f10c0358566152677db51e1e2dc8c\",\"swapFee\":0.000001,\"exchange\":\"balancer\",\"type\":\"balancer-composable-stable\",\"timestamp\":1689615648,\"reserves\":[\"0\",\"0\",\"0\"],\"tokens\":[{\"address\":\"0x02d928e68d8f10c0358566152677db51e1e2dc8c\",\"weight\":333333333333333333,\"swappable\":true},{\"address\":\"0x60d604890feaa0b5460b28a424407c24fe89374a\",\"weight\":333333333333333333,\"swappable\":true},{\"address\":\"0xf951e335afb289353dc249e82926178eac7ded78\",\"weight\":333333333333333333,\"swappable\":true}],\"staticExtra\":\"{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x02d928e68d8f10c0358566152677db51e1e2dc8c00000000000000000000051e\",\"tokenDecimals\":[18,18,18]}\"}"
	var pair = entity.Pool{
		Address:      "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431",
		ReserveUsd:   0,
		AmplifiedTvl: 0,
		SwapFee:      0.000001,
		Exchange:     "balancer",
		Type:         "balancer-composable-stable",
		Timestamp:    1689789778,
		Reserves:     entity.PoolReserves{"2455412046525881020921", "2596148429266323438822175768385755", "3518407624731693170104"},
		Tokens: entity.PoolTokens{
			&entity.PoolToken{
				Address:   "0x60d604890feaa0b5460b28a424407c24fe89374a",
				Name:      "A",
				Symbol:    "",
				Weight:    333333333333333333,
				Swappable: true,
			},
			&entity.PoolToken{
				Address:   "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431",
				Name:      "B",
				Symbol:    "",
				Weight:    333333333333333333,
				Swappable: true,
			},
			&entity.PoolToken{
				Address:   "0xbe9895146f7af43049ca1c1ae358b0541ea49704",
				Name:      "C",
				Symbol:    "",
				Weight:    333333333333333333,
				Swappable: true,
			},
		},
		Extra:       "{\"amplificationParameter\":{\"value\":700000,\"isUpdating\":false,\"precision\":1000},\"scalingFactors\":[1003637040578540860,1000000000000000000,1043680240732074966],\"bptIndex\":1,\"actualSupply\":6105781862789255176406,\"lastJoinExit\":{\"LastJoinExitAmplification\":700000,\"LastPostJoinExitInvariant\":6135006746648647084879},\"rateProviders\":[\"0x60d604890feaa0b5460b28a424407c24fe89374a\",\"0x0000000000000000000000000000000000000000\",\"0x7311e4bb8a72e7b300c5b8bde4de6cdaa822a5b1\"],\"tokensExemptFromYieldProtocolFee\":[false,false,false],\"tokenRateCaches\":[{\"Rate\":1003637040578540860,\"OldRate\":1003554274984131981,\"Duration\":21600,\"Expires\":1689810275},{\"Rate\":null,\"OldRate\":null,\"Duration\":null,\"Expires\":null},{\"Rate\":1043680240732074966,\"OldRate\":1043375386816533719,\"Duration\":21600,\"Expires\":1689810275}],\"protocolFeePercentageCacheSwapType\":0,\"protocolFeePercentageCacheYieldType\":0}",
		StaticExtra: "{\"vaultAddress\":\"0xba12222222228d8ba445958a75a0704d566bf2c8\",\"poolId\":\"0x9001cbbd96f54a658ff4e6e65ab564ded76a543100000000000000000000050a\",\"tokenDecimals\":[18,18,18]}",
		TotalSupply: "2596148429272429220684965023562161",
	}

	var p, err = NewPoolSimulator(pair)
	require.Nil(t, err)
	assert.Equal(t, []string{"0x60d604890feaa0b5460b28a424407c24fe89374a", "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431"}, p.CanSwapTo("0xbe9895146f7af43049ca1c1ae358b0541ea49704"))
	assert.Equal(t, 0, len(p.CanSwapTo("Ax")))

	var tokenAmountIn = pool.TokenAmount{
		Token:  "0x9001cbbd96f54a658ff4e6e65ab564ded76a5431",
		Amount: bignumber.NewBig10("1000000000000000000"),
	}
	var tokenOut = "0xbe9895146f7af43049ca1c1ae358b0541ea49704"
	result, _ := p.CalcAmountOut(tokenAmountIn, tokenOut)
	assert.NotNil(t, result.TokenAmountOut)
	assert.NotNil(t, result.Fee)
	assert.NotNil(t, result.Gas)
	assert.Equal(t, "99832311090", result.TokenAmountOut.Amount.String())
}
