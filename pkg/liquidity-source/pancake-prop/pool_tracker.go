package pancakeprop

import (
	"context"
	"math"
	"math/big"
	"slices"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

// ERC-1967 implementation slot of the pricing proxy.
var implSlot = common.HexToHash("0x360894a13ba1a3210667c828492db98dca3e2076cc3735a920a3ca505d382bbc")

type PoolTracker struct {
	cfg    *Config
	client *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, client *ethrpc.Client) *PoolTracker {
	return &PoolTracker{cfg: cfg, client: client}
}

// GetNewPoolState reads the pair state at one block. The pricing module has no getter for the
// price word, so it is read from storage (mapping slot 4 keyed by pair id).
func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool, _ pool.GetNewPoolStateParams) (entity.Pool, error) {
	var static StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &static); err != nil {
		return p, err
	}
	if len(p.Tokens) != 2 || static.PairID == 0 {
		return p, ErrInvalidState
	}

	eth := t.client.GetETHClient()
	header, err := eth.HeaderByNumber(ctx, nil)
	if err != nil {
		return p, err
	}
	blockHash := header.Hash()
	pricing := common.HexToAddress(static.Pricing)

	impl, err := eth.StorageAtHash(ctx, pricing, implSlot, blockHash)
	if err != nil {
		return p, err
	}
	if !t.supported(common.BytesToAddress(impl)) {
		return p, ErrUnsupportedImpl
	}
	packedRaw, err := eth.StorageAtHash(ctx, pricing, priceSlot(static.PairID), blockHash)
	if err != nil {
		return p, err
	}
	var extra Extra
	extra.Packed.SetBytes(packedRaw)
	if extra.Packed.IsZero() {
		return p, ErrInvalidState
	}

	var maxInput, cum uintPair
	var balances [2]*big.Int
	pairID := new(big.Int).SetUint64(static.PairID)
	req := t.client.NewRequest().SetContext(ctx).SetBlockHash(blockHash)
	req.AddCall(&ethrpc.Call{ABI: pricingABI, Target: static.Pricing, Method: methodPaused}, []any{&extra.Paused})
	req.AddCall(&ethrpc.Call{ABI: pricingABI, Target: static.Pricing, Method: methodGetPairGlobalCfg, Params: []any{pairID}},
		[]any{&maxInput})
	req.AddCall(&ethrpc.Call{ABI: pricingABI, Target: static.Pricing, Method: methodGetPairCumulative, Params: []any{pairID}},
		[]any{&cum})
	for i, token := range p.Tokens {
		req.AddCall(&ethrpc.Call{ABI: abi.Erc20ABI, Target: token.Address, Method: abi.Erc20BalanceOfMethod,
			Params: []any{common.HexToAddress(static.Vault)}}, []any{&balances[i]})
	}
	res, err := req.Aggregate()
	if err != nil {
		return p, err
	}
	if res.BlockNumber == nil || res.BlockNumber.Cmp(header.Number) != 0 {
		return p, ErrInvalidState
	}

	for d := range 2 {
		if maxInput.at(d) == nil || cum.at(d) == nil || balances[d] == nil ||
			extra.MaxInput[d].SetFromBig(maxInput.at(d)) || extra.Cum[d].SetFromBig(cum.at(d)) ||
			extra.MaxInput[d].BitLen() > 128 || extra.Cum[d].BitLen() > 128 {
			return p, ErrInvalidState
		}
	}

	// Expiry is a block number; the price is valid through block `expiry`. Convert it to wall
	// clock from the header time. A price expired at the header lands in the past.
	expiry := int64(unpackPrice(&extra.Packed).expiry)
	if expiry == 0 { // the pricing module never expires a price with expiry 0
		extra.ExpiredAtMs = math.MaxInt64
	} else {
		extra.ExpiredAtMs = uint64(int64(header.Time)*1000 + (expiry-header.Number.Int64())*int64(t.cfg.BlockTimeMs))
	}

	extraBytes, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}
	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{balances[0].String(), balances[1].String()}
	p.BlockNumber = header.Number.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

func (t *PoolTracker) supported(impl common.Address) bool {
	return slices.ContainsFunc(t.cfg.PricingImplementations, func(a string) bool {
		return strings.EqualFold(a, impl.Hex())
	})
}

// uintPair unpacks a two-output (uint256, uint256) view; ethrpc needs a struct for 2+ outputs.
type uintPair struct {
	V0 *big.Int `abi:"v0"`
	V1 *big.Int `abi:"v1"`
}

func (p *uintPair) at(i int) *big.Int {
	if i == 0 {
		return p.V0
	}
	return p.V1
}

// priceSlot is keccak256(pairId . 4), the storage slot of the pair's price word.
func priceSlot(pairID uint64) common.Hash {
	var key [64]byte
	new(uint256.Int).SetUint64(pairID).WriteToArray32((*[32]byte)(key[:32]))
	key[63] = slotPairPrice
	return crypto.Keccak256Hash(key[:])
}
