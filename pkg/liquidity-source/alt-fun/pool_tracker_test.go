package altfun

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const (
	rpcURL           = "https://rpc.hyperliquid.xyz/evm"
	multicallAddress = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

// testPoolJSON is a real pool discovered by the list updater.
const testPoolJSON = `{"address":"0xfe42150691ae59df8b8be440ff9632545aa00000","exchange":"alt-fun","type":"alt-fun","timestamp":1779965481,"reserves":["0","0"],"tokens":[{"address":"0xb88339cb7199b77e23db6e890353e22632ba630f","swappable":true},{"address":"0xfe42150691ae59df8b8be440ff9632545aa00000","swappable":true}],"extra":"{}","staticExtra":"{\"pairAddress\":\"0xc5b60dd9d54ad91bb2365e99964277bcc49e893e\",\"ltAddress\":\"0x18b8539261cf9e760e7fec4a8a73c50f0ae7babe\",\"usdc\":\"0xb88339CB7199b77E23DB6E890353E22632Ba630f\",\"zapAddress\":\"0x693F12E9E6B35b34458793546065E8b08e0299d6\",\"buyFeeBps\":75,\"sellFeeBps\":75,\"basePools\":[\"0x18b8539261cf9e760e7fec4a8a73c50f0ae7babe\"],\"graduationThresholdUsd\":\"9000000000000000000000\"}"}`

type PoolTrackerTestSuite struct {
	suite.Suite
	tracker *PoolTracker
}

func (ts *PoolTrackerTestSuite) SetupSuite() {
	cfg := &Config{
		DexID:                DexType,
		ZapAddress:           "0x693F12E9E6B35b34458793546065E8b08e0299d6",
		BondingAddress:       "0xb68811BcC0e4FcD825aA49F9453b065ddF752FcB",
		FactoryAddress:       "0xd5E5Fef4cFeFb67bbA0aA1dc74B2Cd196B4786AC",
		GlobalStorageAddress: "0xa07d06383c1863c8A54d427aC890643d76cc03ff",
		APIURL:               defaultAPIURL,
		NewPoolLimit:         100,
	}

	client := ethrpc.New(rpcURL).
		SetMulticallContract(common.HexToAddress(multicallAddress))

	tracker, err := NewPoolTracker(cfg, client)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

func (ts *PoolTrackerTestSuite) TestGetNewPoolState() {
	t := ts.T()

	var p entity.Pool
	require.NoError(t, json.Unmarshal([]byte(testPoolJSON), &p))

	updated, err := ts.tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	t.Logf("address:     %s", updated.Address)
	t.Logf("blockNumber: %d", updated.BlockNumber)
	t.Logf("reserves:    %v", updated.Reserves)
	t.Logf("extra:       %s", updated.Extra)

	require.Len(t, updated.Reserves, 2)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))

	t.Logf("lifecycle:     %d", extra.Lifecycle)
	t.Logf("reserveToken:  %s", extra.ReserveToken)
	t.Logf("reserveAsset:  %s", extra.ReserveAsset)
	t.Logf("k:             %s", extra.K)
	t.Logf("tokenBalance:  %s", extra.TokenBalance)
}

func TestPoolTrackerTestSuite(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)
	suite.Run(t, new(PoolTrackerTestSuite))
}
