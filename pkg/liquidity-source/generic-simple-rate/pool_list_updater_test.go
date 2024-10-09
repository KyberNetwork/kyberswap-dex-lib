package generic_simple_rate

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/holiman/uint256"
)

func Test_getNewPool(t *testing.T) {
	rateDefault := big.NewInt(100)

	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io")
	rpcClient.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	config := &Config{
		RateMethod:  "getExchangeRate",
		DefaultRate: rateDefault,
		DexID:       "mkr-sky",
		RateUnit:    big.NewInt(1),
	}

	updater := NewPoolsListUpdater(config, rpcClient)

	poolItem := PoolItem{
		ID:   "0xcf5EA1b38380f6aF39068375516Daf40Ed70D299",
		Type: "mkr-sky",
		Tokens: []entity.PoolToken{
			{
				Address:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
				Swappable: true,
			},
			{
				Address:   "0x56072c95faa701256059aa122697b133aded9279",
				Swappable: true,
			},
		},
	}

	expectedPool := entity.Pool{
		Address:   "0xcf5EA1b38380f6aF39068375516Daf40Ed70D299",
		Exchange:  "mkr-sky",
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{defaultReserves, defaultReserves},
		Tokens: []*entity.PoolToken{
			{
				Address:   "0x9f8f72aa9304c8b593d555f12ef6589cc3a579a2",
				Weight:    1,
				Swappable: true,
			},
			{
				Address:   "0x56072c95faa701256059aa122697b133aded9279",
				Weight:    1,
				Swappable: true,
			},
		},
		Extra: func() string {
			extraBytes, _ := json.Marshal(PoolExtra{
				Rate:       uint256.MustFromBig(rateDefault),
				RateUnit:   uint256.MustFromBig(big.NewInt(1)),
				Paused:     false,
				DefaultGas: DefaultGas,
			})
			return string(extraBytes)
		}(),
	}

	t.Run("test get new pool", func(t *testing.T) {
		pool, _ := updater.getNewPool(&poolItem)
		assert.Equal(t, expectedPool.Address, pool.Address)
		assert.Equal(t, expectedPool.Exchange, pool.Exchange)
		assert.Equal(t, expectedPool.Type, pool.Type)
		assert.Equal(t, expectedPool.Reserves, pool.Reserves)
		assert.Equal(t, expectedPool.Tokens, pool.Tokens)
		assert.Equal(t, expectedPool.Extra, pool.Extra)
	})
}
