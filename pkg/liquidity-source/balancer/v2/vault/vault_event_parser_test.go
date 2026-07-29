package vault

import (
	"context"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/shared"
)

const vaultV2 = "0xBA12222222228d8Ba445958a75a0704d566BF2C8"

func TestDecode(t *testing.T) {
	t.Parallel()

	t.Run("1. Decode pool address from a real Vault Swap log", func(t *testing.T) {
		// Real Balancer V2 Vault Swap log from eth_getLogs, tx
		// 0x313952217900e31ae29f61d6f7fcb5c1406584824aa3f8f1816a03463c76838d (block 25635720).
		// poolId topic1 = 0x3de27efa2f1aa663ae5d458857e731c129069f29000200000000000000000588,
		// so the pool address is the high-order 20 bytes: 0x3de27efa2f1aa663ae5d458857e731c129069f29.
		jsonStr := `[{"removed":false,"logIndex":"0x74","transactionIndex":"0x21","transactionHash":"0x313952217900e31ae29f61d6f7fcb5c1406584824aa3f8f1816a03463c76838d","blockHash":"0x21972b0234c764bcc4db7df588b768a6c488075869778d0e05fdd442640d6e54","blockNumber":"0x1872888","address":"0xba12222222228d8ba445958a75a0704d566bf2c8","data":"0x000000000000000000000000000000000000000000000000932eabea11d74a65000000000000000000000000000000000000000000000000064454de48666cab","topics":["0x2170c741c41531aec20e7c107c24eecfdd15e69c9bb0a8dd37b1840b9e0b207b","0x3de27efa2f1aa663ae5d458857e731c129069f29000200000000000000000588","0x0000000000000000000000007fc66500c84a76ad7e9c93437bfc5ac33e2ddae9","0x0000000000000000000000007f39c581f595b53c5cb19bd0b3f8da6c935e2ca0"]}]`
		var logs []types.Log
		require := assert.New(t)
		_ = json.Unmarshal([]byte(jsonStr), &logs)

		poolDecoder := NewPoolFactory(&Config{Vault: vaultV2})
		addressLogs, err := poolDecoder.Decode(context.Background(), logs)
		require.NoError(err)

		const expected = "0x3de27efa2f1aa663ae5d458857e731c129069f29"
		require.Len(addressLogs[expected], 1)
		require.Equal(uint(0x74), addressLogs[expected][0].Index)
	})

	t.Run("2. Decode target pool (60WETH-40DAI) from its poolId", func(t *testing.T) {
		require := assert.New(t)
		// Real on-chain registered poolId of the Balancer 60WETH-40DAI weighted pool.
		// High-order 20 bytes = 0x0b09dea16768f0799065c475be02919503cb2a35 (the pool address).
		const poolID = "0x0b09dea16768f0799065c475be02919503cb2a350002000000000000000000fe"
		// Derive the expectation from the poolId itself (non-circular vs the decoder).
		expected := strings.ToLower(poolID[:42])

		// Build a Swap log the way the V2 Vault emits it: topic0 = Swap event ID (from the
		// ABI, not hand-typed), topic1 = poolId.
		log := types.Log{
			Address: common.HexToAddress(vaultV2),
			Topics: []common.Hash{
				shared.VaultABI.Events["Swap"].ID,
				common.HexToHash(poolID),
			},
		}

		poolDecoder := NewPoolFactory(&Config{Vault: vaultV2})
		addresses, err := poolDecoder.DecodePoolAddressesFromFactoryLog(context.Background(), log)
		require.NoError(err)
		require.Equal([]string{expected}, addresses)
	})
}
