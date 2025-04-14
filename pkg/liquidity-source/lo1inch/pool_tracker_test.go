package lo1inch

import (
	"context"
	"strings"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	t.Run("1. Log not removed", func(t *testing.T) {
		// eth_getLogs(0xd61ce81fc1f178b1679491cd24fed6c5b11c6c6271acf940cfa623b3b5f4eab4)
		logsStr := `[{"address":"0x111111125421ca6dc452d289314280a0f8842a65","topics":["0xfec331350fce78ba658e082a71da20ac9f8d798a99b3c79681c8440cbfe77e07"],"data":"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b30354100000000000000000000000000000000000000000000000000000000038261cb","blockNumber":"0x1536abb","transactionHash":"0xd61ce81fc1f178b1679491cd24fed6c5b11c6c6271acf940cfa623b3b5f4eab4","transactionIndex":"0xb","blockHash":"0x33a9e4bc780488d6b927a986175b9291b8c2c685f6e37dac04741d07b852d4b5","logIndex":"0x2b","removed":false}]`
		var logs []types.Log
		if err := json.Unmarshal([]byte(logsStr), &logs); err != nil {
			t.Fatal(err)
		}
		tracker := NewPoolTracker()

		poolE := entity.Pool{
			Address: "0x123",
			Extra:   "{\"takeToken0Orders\":[{\"signature\":\"\",\"orderHash\":\"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b303541\",\"remainingMakerAmount\":\"100\",\"makerBalance\":\"0\",\"makerAllowance\":\"0\",\"makerAsset\":\"\",\"takerAsset\":\"\",\"salt\":\"\",\"receiver\":\"\",\"makingAmount\":\"0\",\"takingAmount\":\"0\",\"maker\":\"\",\"extension\":\"\",\"makerTraits\":\"\",\"isMakerContract\":false}],\"takeToken1Orders\":[]}",
		}
		newPool, _ := tracker.GetNewPoolState(context.Background(), poolE, pool.GetNewPoolStateParams{
			Logs: logs,
		})

		assert.Equal(t, "{\"takeToken0Orders\":[{\"signature\":\"\",\"orderHash\":\"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b303541\",\"remainingMakerAmount\":\"58876363\",\"makerBalance\":\"0\",\"makerAllowance\":\"0\",\"makerAsset\":\"\",\"takerAsset\":\"\",\"salt\":\"\",\"receiver\":\"\",\"makingAmount\":\"0\",\"takingAmount\":\"0\",\"maker\":\"\",\"extension\":\"\",\"makerTraits\":\"\",\"isMakerContract\":false}],\"takeToken1Orders\":[]}", newPool.Extra)
	})
	t.Run("2. Log removed", func(t *testing.T) {
		// eth_getLogs(0xd61ce81fc1f178b1679491cd24fed6c5b11c6c6271acf940cfa623b3b5f4eab4)
		logsStr := `[{"address":"0x111111125421ca6dc452d289314280a0f8842a65","topics":["0xfec331350fce78ba658e082a71da20ac9f8d798a99b3c79681c8440cbfe77e07"],"data":"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b30354100000000000000000000000000000000000000000000000000000000038261cb","blockNumber":"0x1536abb","transactionHash":"0xd61ce81fc1f178b1679491cd24fed6c5b11c6c6271acf940cfa623b3b5f4eab4","transactionIndex":"0xb","blockHash":"0x33a9e4bc780488d6b927a986175b9291b8c2c685f6e37dac04741d07b852d4b5","logIndex":"0x2b","removed":true}]`
		var logs []types.Log
		if err := json.Unmarshal([]byte(logsStr), &logs); err != nil {
			t.Fatal(err)
		}
		tracker := NewPoolTracker()

		poolE := entity.Pool{
			Address: "0x123",
			Extra:   "{\"takeToken0Orders\":[{\"signature\":\"\",\"orderHash\":\"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b303541\",\"remainingMakerAmount\":\"100\",\"makerBalance\":\"0\",\"makerAllowance\":\"0\",\"makerAsset\":\"\",\"takerAsset\":\"\",\"salt\":\"\",\"receiver\":\"\",\"makingAmount\":\"0\",\"takingAmount\":\"0\",\"maker\":\"\",\"extension\":\"\",\"makerTraits\":\"\",\"isMakerContract\":false}],\"takeToken1Orders\":[]}",
		}
		newPool, _ := tracker.GetNewPoolState(context.Background(), poolE, pool.GetNewPoolStateParams{
			Logs: logs,
		})

		assert.Equal(t, "{\"takeToken0Orders\":[{\"signature\":\"\",\"orderHash\":\"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b303541\",\"remainingMakerAmount\":\"100\",\"makerBalance\":\"0\",\"makerAllowance\":\"0\",\"makerAsset\":\"\",\"takerAsset\":\"\",\"salt\":\"\",\"receiver\":\"\",\"makingAmount\":\"0\",\"takingAmount\":\"0\",\"maker\":\"\",\"extension\":\"\",\"makerTraits\":\"\",\"isMakerContract\":false}],\"takeToken1Orders\":[]}", newPool.Extra)
	})
	t.Run("2. Log removed + 2 consequences OrderFilled", func(t *testing.T) {
		// 1st log: a removed log
		log1 := `{"address":"0x111111125421ca6dc452d289314280a0f8842a65","topics":["0xfec331350fce78ba658e082a71da20ac9f8d798a99b3c79681c8440cbfe77e07"],"data":"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b30354100000000000000000000000000000000000000000000000000000000038261cb","blockNumber":"0x1536abb","transactionHash":"0xd61ce81fc1f178b1679491cd24fed6c5b11c6c6271acf940cfa623b3b5f4eab4","transactionIndex":"0xb","blockHash":"0x33a9e4bc780488d6b927a986175b9291b8c2c685f6e37dac04741d07b852d4b5","logIndex":"0x2b","removed":true}`
		// 2nd log: a valid log with same block and higher logIndex with remainingMakerAmount 58876363
		log2 := strings.ReplaceAll(log1, `"removed":true`, `"removed":false`)
		// 3rd log: a valid log with same block but lower logIndex with remainingMakerAmount 58876362
		log3 := strings.ReplaceAll(strings.ReplaceAll(log2, "0x2b", "0x2a"), `38261cb`, `38261ca`)
		logsStr := "[" + log1 + "," + log2 + "," + log3 + "]"
		var logs []types.Log
		if err := json.Unmarshal([]byte(logsStr), &logs); err != nil {
			t.Fatal(err)
		}
		poolExtraBefore := "{\"takeToken0Orders\":[{\"signature\":\"\",\"orderHash\":\"0x1241aa182441c83bb5f5ed094721ce88e01960287561be51d4b0f3248b303541\",\"remainingMakerAmount\":\"123456789\",\"makerBalance\":\"0\",\"makerAllowance\":\"0\",\"makerAsset\":\"\",\"takerAsset\":\"\",\"salt\":\"\",\"receiver\":\"\",\"makingAmount\":\"0\",\"takingAmount\":\"0\",\"maker\":\"\",\"extension\":\"\",\"makerTraits\":\"\",\"isMakerContract\":false}],\"takeToken1Orders\":[]}"

		// 123456789 -> 58876363 (2nd log)
		poolExtraAfter := strings.ReplaceAll(poolExtraBefore, `123456789`, `58876363`)
		poolE := entity.Pool{
			Address: "0x123",
			Extra:   poolExtraBefore,
		}

		tracker := NewPoolTracker()
		newPool, _ := tracker.GetNewPoolState(context.Background(), poolE, pool.GetNewPoolStateParams{
			Logs: logs,
		})

		assert.Equal(t, poolExtraAfter, newPool.Extra)
	})

}
