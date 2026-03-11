package lunarbase

import (
	"context"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolSimulator struct {
	pool.Pool

	exactQuoter    *gethclient.Client
	chainID        valueobject.ChainID
	periphery      string
	permit2        string
	wrappedNative  string
	rawTokenX      string
	rawTokenY      string
	priceX96       *uint256.Int
	fee            *uint256.Int
	latestBlock    uint64
	concentrationK uint32
	paused         bool
	reserves       []*uint256.Int
	gas            int64
}

type quoteExactInParams struct {
	TokenIn  common.Address
	TokenOut common.Address
	AmountIn *big.Int
}

type SwapInfo struct {
	NextPX96 *uint256.Int
}

var _ = pool.RegisterFactory(DexType, NewPoolSimulatorFactory)

func NewPoolSimulatorFactory(params pool.FactoryParams) (*PoolSimulator, error) {
	return newPoolSimulator(params.EntityPool, params.ChainID, params.EthClient)
}

func NewPoolSimulator(entityPool entity.Pool, chainID valueobject.ChainID) (*PoolSimulator, error) {
	return newPoolSimulator(entityPool, chainID, nil)
}

func newPoolSimulator(
	entityPool entity.Pool,
	chainID valueobject.ChainID,
	ethClient ethereum.ContractCaller,
) (*PoolSimulator, error) {
	if chainID == 0 {
		chainID = valueobject.ChainIDBase
	}

	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(entityPool.StaticExtra), &staticExtra); err != nil {
		return nil, err
	}

	var exactQuoter *gethclient.Client
	if clientWithRPC, ok := ethClient.(interface{ Client() *rpc.Client }); ok && clientWithRPC.Client() != nil {
		exactQuoter = gethclient.New(clientWithRPC.Client())
	}

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: pool.PoolInfo{
				Address:     entityPool.Address,
				Exchange:    entityPool.Exchange,
				Type:        entityPool.Type,
				Tokens:      lo.Map(entityPool.Tokens, func(item *entity.PoolToken, _ int) string { return item.Address }),
				Reserves:    lo.Map(entityPool.Reserves, func(item string, _ int) *big.Int { return bignumber.NewBig(item) }),
				BlockNumber: entityPool.BlockNumber,
			},
		},
		exactQuoter:    exactQuoter,
		chainID:        chainID,
		periphery:      staticExtra.PeripheryAddress,
		permit2:        lo.Ternary(staticExtra.Permit2Address != "", staticExtra.Permit2Address, defaultPermit2Address),
		wrappedNative:  staticExtra.WrappedNative,
		rawTokenX:      staticExtra.RawTokenX,
		rawTokenY:      staticExtra.RawTokenY,
		priceX96:       extra.PX96,
		fee:            uint256.NewInt(extra.Fee),
		latestBlock:    extra.LatestUpdateBlock,
		concentrationK: extra.ConcentrationK,
		paused:         extra.Paused,
		reserves: lo.Map(entityPool.Reserves, func(item string, _ int) *uint256.Int {
			return big256.New(item)
		}),
		gas: defaultGas,
	}, nil
}

func (s *PoolSimulator) CalcAmountOut(params pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenOut)
	if indexIn < 0 || indexOut < 0 {
		return nil, ErrInvalidToken
	}
	if s.paused {
		return nil, ErrPoolPaused
	}
	if s.priceX96 == nil || s.priceX96.IsZero() {
		return nil, ErrZeroPrice
	}

	amountIn, overflow := uint256.FromBig(params.TokenAmountIn.Amount)
	if overflow || amountIn.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	if s.exactQuoter != nil {
		return s.calcAmountOutExact(params, indexIn, indexOut, amountIn)
	}

	return s.calcAmountOutApprox(params, indexIn, indexOut, amountIn)
}

func (s *PoolSimulator) calcAmountOutExact(
	params pool.CalcAmountOutParams,
	indexIn, indexOut int,
	amountIn *uint256.Int,
) (*pool.CalcAmountOutResult, error) {
	method := "quoteXToY"
	if indexIn == 1 {
		method = "quoteYToX"
	}

	data, err := coreABI.Pack(method, amountIn.ToBig())
	if err != nil {
		return nil, err
	}

	overrides := map[common.Address]gethclient.OverrideAccount{
		common.HexToAddress(s.Info.Address): {
			StateDiff: map[common.Hash]common.Hash{
				pmmSlotState:    s.packStateSlot(),
				pmmSlotReserves: s.packReservesSlot(),
			},
		},
	}

	result, err := s.exactQuoter.CallContract(
		context.Background(),
		ethereum.CallMsg{
			To:   lo.ToPtr(common.HexToAddress(s.Info.Address)),
			Data: data,
		},
		nil,
		&overrides,
	)
	if err != nil {
		return nil, ErrQuoteFailed
	}

	unpacked, err := coreABI.Unpack(method, result)
	if err != nil {
		return nil, err
	}
	if len(unpacked) != 3 {
		return nil, ErrQuoteFailed
	}

	amountOut, ok := unpacked[0].(*big.Int)
	if !ok || amountOut == nil || amountOut.Sign() <= 0 {
		return nil, ErrInsufficientLiquidity
	}

	pNext, ok := unpacked[1].(*big.Int)
	if !ok || pNext == nil || pNext.Sign() <= 0 {
		return nil, ErrQuoteFailed
	}

	feeAmount, ok := unpacked[2].(*big.Int)
	if !ok || feeAmount == nil || feeAmount.Sign() < 0 {
		return nil, ErrInsufficientLiquidity
	}
	if amountOut.Cmp(s.reserves[indexOut].ToBig()) > 0 {
		return nil, ErrInsufficientLiquidity
	}

	nextPX96 := uint256.MustFromBig(pNext)

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut,
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: feeAmount,
		},
		Gas: s.gas,
		SwapInfo: SwapInfo{
			NextPX96: nextPX96,
		},
	}, nil
}

func (s *PoolSimulator) calcAmountOutApprox(
	params pool.CalcAmountOutParams,
	indexIn, indexOut int,
	amountIn *uint256.Int,
) (*pool.CalcAmountOutResult, error) {
	var grossOut uint256.Int
	if indexIn == 0 {
		big256.MulDivDown(&grossOut, amountIn, s.priceX96, q96)
	} else {
		big256.MulDivDown(&grossOut, amountIn, q96, s.priceX96)
	}

	var feeAmount uint256.Int
	big256.MulDivDown(&feeAmount, &grossOut, s.fee, feePrecision)

	amountOut := new(uint256.Int).Sub(&grossOut, &feeAmount)
	if amountOut.IsZero() || amountOut.Gt(s.reserves[indexOut]) {
		return nil, ErrInsufficientLiquidity
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: amountOut.ToBig(),
		},
		Fee: &pool.TokenAmount{
			Token:  params.TokenOut,
			Amount: feeAmount.ToBig(),
		},
		Gas: s.gas,
	}, nil
}

func (s *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *s
	if s.priceX96 != nil {
		cloned.priceX96 = new(uint256.Int).Set(s.priceX96)
	}
	cloned.reserves = lo.Map(s.reserves, func(item *uint256.Int, _ int) *uint256.Int {
		if item == nil {
			return nil
		}

		return new(uint256.Int).Set(item)
	})
	cloned.Info.Reserves = lo.Map(cloned.reserves, func(item *uint256.Int, _ int) *big.Int {
		return item.ToBig()
	})

	return &cloned
}

func (s *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	indexIn, indexOut := s.GetTokenIndex(params.TokenAmountIn.Token), s.GetTokenIndex(params.TokenAmountOut.Token)
	if indexIn < 0 || indexOut < 0 {
		return
	}

	inAmount := uint256.MustFromBig(params.TokenAmountIn.Amount)
	outAmount := uint256.MustFromBig(params.TokenAmountOut.Amount)

	s.reserves[indexIn] = new(uint256.Int).Add(s.reserves[indexIn], inAmount)
	s.reserves[indexOut] = new(uint256.Int).Sub(s.reserves[indexOut], outAmount)
	s.Info.Reserves[indexIn] = s.reserves[indexIn].ToBig()
	s.Info.Reserves[indexOut] = s.reserves[indexOut].ToBig()

	if swapInfo, ok := params.SwapInfo.(SwapInfo); ok && swapInfo.NextPX96 != nil {
		s.priceX96 = new(uint256.Int).Set(swapInfo.NextPX96)
	}
}

func (s *PoolSimulator) GetMetaInfo(tokenIn, tokenOut string) any {
	return PoolMeta{
		BlockNumber:     s.Info.BlockNumber,
		RouterAddress:   s.periphery,
		Permit2Address:  s.permit2,
		ApprovalAddress: s.GetApprovalAddress(tokenIn, tokenOut),
	}
}

func (s *PoolSimulator) GetApprovalAddress(tokenIn, _ string) string {
	if valueobject.IsNative(tokenIn) || valueobject.IsWrappedNative(tokenIn, s.chainID) {
		return ""
	}

	return s.permit2
}

func (s *PoolSimulator) CanSwapTo(address string) []string {
	if s.GetTokenIndex(address) < 0 {
		return nil
	}

	return lo.Filter(s.Info.Tokens, func(token string, _ int) bool {
		return !strings.EqualFold(token, address)
	})
}

func (s *PoolSimulator) CanSwapFrom(address string) []string {
	return s.CanSwapTo(address)
}

func (s *PoolSimulator) rawTokenAddress(index int) common.Address {
	if index == 0 {
		return common.HexToAddress(s.rawTokenX)
	}

	return common.HexToAddress(s.rawTokenY)
}

func (s *PoolSimulator) packStateSlot() common.Hash {
	packed := uint256.NewInt(0)
	if s.priceX96 != nil {
		packed = new(uint256.Int).Set(s.priceX96)
	}

	packed = new(uint256.Int).Add(packed, new(uint256.Int).Lsh(uint256.NewInt(s.fee.Uint64()), 160))
	packed = new(uint256.Int).Add(packed, new(uint256.Int).Lsh(uint256.NewInt(s.latestBlock), 208))

	return common.BytesToHash(common.LeftPadBytes(packed.Bytes(), 32))
}

func (s *PoolSimulator) packReservesSlot() common.Hash {
	packed := new(uint256.Int).Set(s.reserves[0])
	packed = new(uint256.Int).Add(packed, new(uint256.Int).Lsh(new(uint256.Int).Set(s.reserves[1]), 112))
	packed = new(uint256.Int).Add(packed, new(uint256.Int).Lsh(uint256.NewInt(uint64(s.concentrationK)), 224))

	return common.BytesToHash(common.LeftPadBytes(packed.Bytes(), 32))
}
