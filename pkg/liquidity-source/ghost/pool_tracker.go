package ghost

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// WILDCARD_RECIPIENT = bytes32(type(uint256).max) = 0xfff…fff
var wildcardRecipient = [32]byte{
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
	0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,
}

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Start getting new state of pool", DexType)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return entity.Pool{}, err
	}

	extra, blockNumber, err := t.fetchState(ctx, p, staticExtra)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	if len(p.Tokens) >= 2 {
		reserves := make(entity.PoolReserves, len(p.Reserves))
		copy(reserves, p.Reserves)
		reserves[1] = extra.Reserve.String()
		p.Reserves = reserves
	}

	logger.WithFields(logger.Fields{
		"exchange": p.Exchange,
		"address":  p.Address,
	}).Infof("[%s] Finish getting new state of pool", DexType)

	return p, nil
}

type storedQuoteResult struct {
	MaxFee     *big.Int
	HalfAmount *big.Int
	IssuedAt   *big.Int
	Expiry     *big.Int
}

func (t *PoolTracker) fetchState(
	ctx context.Context,
	p entity.Pool,
	se StaticExtra,
) (Extra, uint64, error) {
	feeContract, err := t.resolveFeeContract(ctx, se)
	if err != nil {
		return Extra{}, 0, err
	}

	var (
		immutableMaxFee     *big.Int
		immutableHalfAmount *big.Int
		standing            storedQuoteResult
		reserve             *big.Int
	)

	outputToken := ""
	if len(p.Tokens) >= 2 {
		outputToken = p.Tokens[1].Address
	}

	targetRouterAddr := common.HexToAddress(se.TargetRouter)
	feeTarget := strings.ToLower(feeContract.Hex())

	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    feeABI,
		Target: feeTarget,
		Method: feeMethodMaxFee,
	}, []any{&immutableMaxFee})

	req.AddCall(&ethrpc.Call{
		ABI:    feeABI,
		Target: feeTarget,
		Method: feeMethodHalfAmount,
	}, []any{&immutableHalfAmount})

	req.AddCall(&ethrpc.Call{
		ABI:    feeABI,
		Target: feeTarget,
		Method: feeMethodQuotes,
		Params: []any{se.LocalDomain, wildcardRecipient},
	}, []any{&standing})

	req.AddCall(&ethrpc.Call{
		ABI:    utilabi.Erc20ABI,
		Target: outputToken,
		Method: utilabi.Erc20BalanceOfMethod,
		Params: []any{targetRouterAddr},
	}, []any{&reserve})

	resp, err := req.Aggregate()
	if err != nil {
		return Extra{}, 0, err
	}

	blockNumber := uint64(0)
	if resp.BlockNumber != nil {
		blockNumber = resp.BlockNumber.Uint64()
	}

	effectiveMaxFee := immutableMaxFee
	effectiveHalfAmount := immutableHalfAmount

	if standing.Expiry != nil && standing.Expiry.Sign() > 0 {
		now := time.Now().Unix()
		if standing.Expiry.Int64() > now {
			effectiveMaxFee = standing.MaxFee
			effectiveHalfAmount = standing.HalfAmount
		}
	}

	return Extra{
		MaxFee:     effectiveMaxFee,
		HalfAmount: effectiveHalfAmount,
		Reserve:    reserve,
	}, blockNumber, nil
}

func (t *PoolTracker) resolveFeeContract(ctx context.Context, se StaticExtra) (common.Address, error) {
	var feeRecipient common.Address
	_, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: strings.ToLower(se.SourceRouter),
		Method: "feeRecipient",
	}, []any{&feeRecipient}).Call()
	if err != nil {
		return common.Address{}, fmt.Errorf("ghost: feeRecipient() call failed: %w", err)
	}

	if feeRecipient == (common.Address{}) {
		return common.Address{}, ErrNoFeeContract
	}

	var recipientFeeType uint8
	_, err = t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routingFeeABI,
		Target: strings.ToLower(feeRecipient.Hex()),
		Method: "feeType",
	}, []any{&recipientFeeType}).Call()
	if err != nil {
		return common.Address{}, fmt.Errorf("ghost: feeType() on feeRecipient failed: %w", err)
	}

	var (
		resolved            common.Address
		targetRouterAddr    = common.HexToAddress(se.TargetRouter)
		targetRouterBytes32 = common.BytesToHash(targetRouterAddr.Bytes())
	)

	switch recipientFeeType {
	case feeTypeCrossCollateralRouting:
		_, err = t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
			ABI:    routingFeeABI,
			Target: strings.ToLower(feeRecipient.Hex()),
			Method: "feeContracts",
			Params: []any{se.LocalDomain, targetRouterBytes32},
		}, []any{&resolved}).Call()
		if err != nil {
			return common.Address{}, fmt.Errorf("ghost: feeContracts() call failed: %w", err)
		}

		if resolved == (common.Address{}) {
			_, err = t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
				ABI:    routingFeeABI,
				Target: strings.ToLower(feeRecipient.Hex()),
				Method: "feeContracts",
				Params: []any{se.LocalDomain, defaultRouterKey},
			}, []any{&resolved}).Call()
			if err != nil {
				return common.Address{}, fmt.Errorf("ghost: feeContracts() fallback call failed: %w", err)
			}
		}

	case feeTypeOffchainQuotedLinear:
		resolved = feeRecipient

	default:
		return common.Address{}, fmt.Errorf("%w: feeRecipient has feeType %d", ErrUnsupportedFeeType, recipientFeeType)
	}

	if resolved == (common.Address{}) {
		return common.Address{}, ErrNoFeeContract
	}

	var resolvedFeeType uint8
	_, err = t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    feeABI,
		Target: strings.ToLower(resolved.Hex()),
		Method: "feeType",
	}, []any{&resolvedFeeType}).Call()
	if err != nil {
		return common.Address{}, fmt.Errorf("ghost: feeType() on resolved fee contract failed: %w", err)
	}

	if resolvedFeeType != feeTypeOffchainQuotedLinear {
		return common.Address{}, fmt.Errorf("%w: got %d", ErrUnsupportedFeeType, resolvedFeeType)
	}

	return resolved, nil
}
