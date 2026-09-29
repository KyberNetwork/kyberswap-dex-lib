package gblin

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidToken
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	extra, blockNumber, err := fetchState(ctx, t.ethrpcClient, p.Address, &staticExtra, t.config.MulticallAddress,
		overrides)
	if err != nil {
		logger.WithFields(logger.Fields{"dexId": t.config.DexID, "address": p.Address, "error": err}).
			Error("failed to fetch vault state")
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	// Reserves are informational: the WETH value held by the vault and the share supply. Mint capacity is not
	// bounded by either.
	navEth, supply := reserveZero, reserveZero
	if extra.NavEth != nil {
		navEth = extra.NavEth.Dec()
	}
	if extra.Supply != nil {
		supply = extra.Supply.Dec()
	}
	p.Reserves = entity.PoolReserves{navEth, supply}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// fetchState reads, in one aggregate pinned to one block, every input GBLIN.buyGBLINWithWeth prices a mint from
// and every condition under which it reverts.
func fetchState(
	ctx context.Context,
	client *ethrpc.Client,
	vault string,
	staticExtra *StaticExtra,
	multicall string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (Extra, uint64, error) {
	var (
		supply, totalEthValue, managementFeeBps, lastAccrual, ethBalance, timestamp *big.Int
		navReliable                                                                 bool
		fees                                                                        struct {
			ProtocolFee    *big.Int
			StabilityFee   *big.Int
			MinDeposit     *big.Int
			OracleAge      *big.Int
			OracleAgeTrade *big.Int
			SellCooldown   *big.Int
			BasketCap      *big.Int
		}
		round struct {
			RoundId         *big.Int
			Answer          *big.Int
			StartedAt       *big.Int
			UpdatedAt       *big.Int
			AnsweredInRound *big.Int
		}
	)

	vaultAddr := common.HexToAddress(vault)
	req := client.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	req.AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: vaultMethodTotalSupply}, []any{&supply})
	req.AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: vaultMethodTotalEthValue,
		Params: []any{big.NewInt(0)}}, []any{&totalEthValue})
	req.AddCall(&ethrpc.Call{ABI: vaultABI, Target: vault, Method: vaultMethodIsNavReliable}, []any{&navReliable})
	req.AddCall(&ethrpc.Call{ABI: lensABI, Target: staticExtra.Lens, Method: lensMethodManagementFeeBps,
		Params: []any{vaultAddr}}, []any{&managementFeeBps})
	req.AddCall(&ethrpc.Call{ABI: lensABI, Target: staticExtra.Lens, Method: lensMethodLastManagementFeeAccrual,
		Params: []any{vaultAddr}}, []any{&lastAccrual})
	req.AddCall(&ethrpc.Call{ABI: lensABI, Target: staticExtra.Lens, Method: lensMethodConfigFees,
		Params: []any{vaultAddr}}, []any{&fees})
	req.AddCall(&ethrpc.Call{ABI: multicall3ABI, Target: multicall, Method: multicallMethodGetEthBalance,
		Params: []any{vaultAddr}}, []any{&ethBalance})
	req.AddCall(&ethrpc.Call{ABI: multicall3ABI, Target: multicall, Method: multicallMethodGetCurrentBlockTimestamp},
		[]any{&timestamp})
	if staticExtra.SequencerFeed != "" {
		req.AddCall(&ethrpc.Call{ABI: aggregatorV3ABI, Target: staticExtra.SequencerFeed,
			Method: aggregatorMethodLatestRoundData}, []any{&round})
	}

	// TryBlockAndAggregate: totalEthValue reverts while a price feed is too old to convert a balance, and that
	// must mark the NAV unreliable rather than fail the refresh. It also returns the block the reads were made at.
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return Extra{}, 0, err
	}
	ok := func(i int) bool { return i < len(resp.Result) && resp.Result[i] }

	if !ok(0) || !ok(3) || !ok(4) || !ok(5) || !ok(7) || supply == nil || managementFeeBps == nil ||
		lastAccrual == nil || fees.ProtocolFee == nil || fees.StabilityFee == nil || fees.MinDeposit == nil ||
		timestamp == nil {
		return Extra{}, 0, ErrStateUnavailable
	}

	extra := Extra{
		Supply:           uint256.MustFromBig(supply),
		LastAccrual:      lastAccrual.Uint64(),
		ManagementFeeBps: managementFeeBps.Uint64(),
		ProtocolFeeBps:   fees.ProtocolFee.Uint64(),
		StabilityFeeBps:  fees.StabilityFee.Uint64(),
		MinDeposit:       uint256.MustFromBig(fees.MinDeposit),
		Timestamp:        timestamp.Uint64(),
	}

	if ok(1) && totalEthValue != nil {
		nav := uint256.MustFromBig(totalEthValue)
		if ok(6) && ethBalance != nil {
			nav.Add(nav, uint256.MustFromBig(ethBalance))
		}
		extra.NavEth = nav
		extra.NavReliable = ok(2) && navReliable
	}

	extra.SequencerUp = staticExtra.SequencerFeed == "" || (ok(8) && sequencerUp(round.Answer, round.StartedAt,
		round.UpdatedAt, extra.Timestamp))

	var blockNumber uint64
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}
	return extra, blockNumber, nil
}

// sequencerUp mirrors GBLIN._checkSequencer: the feed reports 1 while the sequencer is down, and mints stay
// refused for an hour after it comes back up.
func sequencerUp(answer, startedAt, updatedAt *big.Int, now uint64) bool {
	if answer == nil || startedAt == nil || updatedAt == nil {
		return false
	}
	if answer.Cmp(big.NewInt(1)) == 0 || startedAt.Sign() == 0 || updatedAt.Sign() == 0 {
		return false
	}
	if !startedAt.IsUint64() {
		return false
	}
	started := startedAt.Uint64()
	return started <= now && now-started > sequencerGracePeriod
}
