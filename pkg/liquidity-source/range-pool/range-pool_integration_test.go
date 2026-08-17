package rangepool

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/suite"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// Live RPC integration tests against the Range Pool deployment on Ethereum mainnet.
//
// Committed but skipped on CI (test.SkipCI); they run only on developer machines with
// a mainnet RPC. Point at your own archive-capable endpoint via:
//
//	export ETH_RPC_URL="https://eth-mainnet.g.alchemy.com/v2/<KEY>"
//	# in this repo the ParaSwap creds are the easiest source:
//	export ETH_RPC_URL=$(grep -m1 PONDER_RPC_URL_1 ../../../../ranges-stats/.env | sed 's/^[^=]*=//' | cut -d, -f1)
//
// Falls back to KyberSwap's public mainnet endpoint (fine here: we only pin reads to
// the just-tracked block, which a full node still serves).
const (
	mainnetDefaultRPC = "https://ethereum-rpc.kyberswap.com"
	mainnetMulticall3 = "0xcA11bde05977b3631167028862bE2a173976CA11"

	// Live pools (reference §1). Low-TVL — mind the fact-balance cap when sizing swaps.
	romeUSDTPool  = "0xaf037e69f0fa8d1633443cc0c67d0b73e3694b36" // 2-token 50/50
	topCryptoPool = "0x67c02fc8f5a4140077999014efa7fe9d0ee2f29b" // 8-token (WETH at index 6)
)

type IntegrationSuite struct {
	suite.Suite

	client  *ethrpc.Client
	lister  *PoolsListUpdater
	tracker *PoolTracker
}

func (ts *IntegrationSuite) SetupSuite() {
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		rpcURL = mainnetDefaultRPC
	}
	ts.client = ethrpc.New(rpcURL).SetMulticallContract(common.HexToAddress(mainnetMulticall3))

	cfg := &Config{
		DexID:          DexType,
		FactoryAddress: FactoryAddress.Hex(),
		NewPoolLimit:   100,
	}
	ts.lister = NewPoolsListUpdater(cfg, ts.client)

	tracker, err := NewPoolTracker(cfg, ts.client)
	ts.Require().NoError(err)
	ts.tracker = tracker
}

// discoverAll pages through the factory and returns every pool keyed by lowercased
// address.
func (ts *IntegrationSuite) discoverAll() map[string]entity.Pool {
	all := make(map[string]entity.Pool)
	var meta []byte
	for round := 0; round < 20; round++ {
		pools, next, err := ts.lister.GetNewPools(context.Background(), meta)
		ts.Require().NoError(err)
		for _, p := range pools {
			all[strings.ToLower(p.Address)] = p
		}
		if len(pools) == 0 {
			break
		}
		meta = next
	}
	return all
}

// TestListPools_IncludesKnownPools verifies on-chain discovery from the factory
// enumerates at least the two known live pools with well-formed metadata.
func (ts *IntegrationSuite) TestListPools_IncludesKnownPools() {
	all := ts.discoverAll()
	ts.Require().GreaterOrEqual(len(all), 2, "factory should expose at least the 2 known pools")

	for _, addr := range []string{romeUSDTPool, topCryptoPool} {
		p, ok := all[addr]
		ts.Require().Truef(ok, "known pool %s not discovered", addr)
		ts.Require().Equal(DexType, p.Type)
		ts.Require().Equal(DexType, p.Exchange)
		ts.Require().NotEmpty(p.Tokens)
		ts.Require().Len(p.Reserves, len(p.Tokens))

		var se StaticExtra
		ts.Require().NoError(json.Unmarshal([]byte(p.StaticExtra), &se))
		ts.Require().Len(se.NormalizedWeights, len(p.Tokens),
			"one normalized weight per token")
		ts.Require().Len(se.DecimalScalingFactors, len(p.Tokens),
			"one scaling factor per token")
	}

	ts.Require().Len(all[romeUSDTPool].Tokens, 2, "ROME/USDT is a 2-token pool")
	ts.Require().Len(all[topCryptoPool].Tokens, 8, "TOP CRYPTO is an 8-token pool")
}

// TestTrack_ROMEUSDT and TestTrack_TopCrypto: discover then track each live pool, and
// assert the tracked Extra equals a direct getRangePoolDynamicData read at the SAME
// block — virtual + fact balances to the wei, and all six liveness flags decoded.
func (ts *IntegrationSuite) TestTrack_ROMEUSDT()  { ts.assertTrackedMatchesChain(romeUSDTPool, 2) }
func (ts *IntegrationSuite) TestTrack_TopCrypto() { ts.assertTrackedMatchesChain(topCryptoPool, 8) }

func (ts *IntegrationSuite) assertTrackedMatchesChain(poolAddr string, wantTokens int) {
	seed, ok := ts.discoverAll()[poolAddr]
	ts.Require().Truef(ok, "pool %s not discovered", poolAddr)

	tracked, err := ts.tracker.GetNewPoolState(context.Background(), seed, pool.GetNewPoolStateParams{})
	ts.Require().NoError(err)

	var extra Extra
	ts.Require().NoError(json.Unmarshal([]byte(tracked.Extra), &extra))

	ts.Require().Len(extra.VirtualBalances, wantTokens)
	ts.Require().Len(extra.BalancesLiveScaled18, wantTokens)
	ts.Require().Len(extra.DecimalScalingFactors, wantTokens)
	ts.Require().Len(extra.TokenRates, wantTokens)
	ts.Require().NotNil(extra.StaticSwapFeePercentage)
	ts.Require().NotNil(extra.AggregateSwapFeePercentage)

	// Direct read pinned to the tracked block — the ground truth.
	onchain := ts.readDynamicDataAt(poolAddr, tracked.BlockNumber)

	for i := 0; i < wantTokens; i++ {
		ts.Require().Equalf(onchain.VirtualBalances[i].String(), extra.VirtualBalances[i].String(),
			"virtualBalances[%d] mismatch", i)
		ts.Require().Equalf(onchain.BalancesLiveScaled18[i].String(), extra.BalancesLiveScaled18[i].String(),
			"balancesLiveScaled18[%d] mismatch", i)
	}

	// All six liveness flags decode and match the chain.
	ts.Require().Equal(onchain.IsPoolRegistered, extra.IsPoolRegistered)
	ts.Require().Equal(onchain.IsPoolInitialized, extra.IsPoolInitialized)
	ts.Require().Equal(onchain.IsPoolPaused, extra.IsPoolPaused)
	ts.Require().Equal(onchain.IsPoolInRecoveryMode, extra.IsPoolInRecoveryMode)
	ts.Require().Equal(onchain.IsVaultPaused, extra.IsVaultPaused)
	ts.Require().Equal(onchain.IsHookStopped, extra.IsHookStopped)

	// A live pool must expose non-zero reserves (fact balances); the tracker mirrors
	// balancesRaw into entity.Pool.Reserves.
	if extra.isLive() {
		ts.Require().Len(tracked.Reserves, wantTokens)
		nonZero := false
		for _, r := range tracked.Reserves {
			if r != "0" {
				nonZero = true
			}
		}
		ts.Require().True(nonZero, "a live pool should have non-zero reserves")
	}

	ts.T().Logf("[%s] block=%d live=%v virtual0=%s factLive0=%s fee=%s aggFee=%s",
		poolAddr, tracked.BlockNumber, extra.isLive(),
		extra.VirtualBalances[0], extra.BalancesLiveScaled18[0],
		extra.StaticSwapFeePercentage, extra.AggregateSwapFeePercentage)
}

func (ts *IntegrationSuite) readDynamicDataAt(poolAddr string, blockNumber uint64) *rangePoolDynamicDataABI {
	var dyn rangePoolDynamicDataResult
	req := ts.client.NewRequest().SetContext(context.Background()).
		SetBlockNumber(new(big.Int).SetUint64(blockNumber))
	req.AddCall(&ethrpc.Call{
		ABI:    rangePoolABI,
		Target: poolAddr,
		Method: poolMethodGetDynamicData,
	}, []any{&dyn})
	_, err := req.Call()
	ts.Require().NoError(err)
	return &dyn.Data
}

func TestIntegrationSuite(t *testing.T) {
	test.SkipCI(t)
	suite.Run(t, new(IntegrationSuite))
}
