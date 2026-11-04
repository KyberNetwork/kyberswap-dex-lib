package nami

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

//go:embed abi.json
var pluginJSON []byte

var namiPluginABI = func() abi.ABI {
	result, err := abi.JSON(bytes.NewReader(pluginJSON))
	if err != nil {
		panic(err)
	}
	return result
}()

const (
	// Per-hop afterSwap overhead, deliberately an overestimate (under-estimating
	// misprices routes). Measured on 134 router swaps on Ethereum and Base:
	// 38.9k-54.6k on the no-backrun, reverted-check and route-construction
	// branches, and 104.7k on one Base reverted-check pool. The external
	// arbitrage branch can cost more; callGasBudget is a ceiling, not consumption.
	MEVXGasEstimate uint64 = 110_000
	// afterSwap reverts when gasleft() is below minGasLeft, and the user's gas
	// limit comes from our estimate, so minGasLeft is added to the estimate as
	// headroom. Above this value the pool is not quoted: the contract allows up
	// to 5M, which would make every hop of a route carry a huge gas limit.
	MaxMinGasLeft uint64 = 1_000_000
	// Pool-service polls plugin state every 10s. Stop routing after a minute
	// without a recent chain snapshot; an idle volatility timepoint is valid.
	MaxSnapshotAge uint64 = 60
)

// Config is the fee override state described by the Nami plugin.
// https://gist.github.com/codeislight1/78ddfe3095de78fef05ae215b996b641
// Fee sources are external arbitrary contracts, so enabled sources are rejected until supported.
type Config struct {
	Floor, Cap, Premium0For1, Premium1For0 uint16
	DecayWindow                            uint32
	MaxDiscountBps                         uint16
	Fee0For1, Fee1For0                     uint16
	ExpiresAt                              uint64
	MEVXGas                                uint64
	Unsupported                            string
	SnapshotTimestamp                      uint64
	MinGasLeft, CallGasBudget              uint64
}

type namiFeeBounds struct {
	Floor, Cap, Premium0For1, Premium1For0 uint16
	DecayWindow                            uint32
	MaxDiscountBps                         uint16
}
type namiAntiSniper struct {
	FeeReceiver        common.Address
	Period             uint32
	StartingFee        *big.Int
	DisabledPriorStart bool
	EaseIn, EaseOut    uint16
}

type namiFeeOverride struct {
	Fee0For1, Fee1For0 uint16
	ExpiresAt          *big.Int
}

func Fetch(ctx context.Context, client *ethrpc.Client, plugin common.Address, block *big.Int, overrides map[common.Address]gethclient.OverrideAccount, now uint32) (*Config, error) {
	if plugin == (common.Address{}) {
		return nil, errors.New("nami plugin is missing")
	}
	target := hexutil.Encode(plugin[:])
	var boundsOut struct{ Bounds namiFeeBounds }
	var overrideOut struct{ OverrideState namiFeeOverride }
	var sourceState struct {
		Source  common.Address
		Enabled bool
	}
	var antiSniperOut struct{ Config namiAntiSniper }
	var startTimestamp uint64
	var mevxRouter common.Address
	var minGasLeft, callGasBudget *big.Int
	req := client.NewRequest().SetContext(ctx).SetBlockNumber(block).SetOverrides(overrides)
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "feeBounds"}, []any{&boundsOut})
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "feeOverride"}, []any{&overrideOut})
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "feeSource"}, []any{&sourceState})
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "getMevxRouter"}, []any{&mevxRouter})
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "getMinGasLeft"}, []any{&minGasLeft})
	req.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "getCallGasBudget"}, []any{&callGasBudget})
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	bounds, override := boundsOut.Bounds, overrideOut.OverrideState
	if override.ExpiresAt == nil || minGasLeft == nil || callGasBudget == nil {
		return nil, errors.New("failed to read Nami fee state")
	}
	var unsupported string
	if sourceState.Enabled {
		unsupported = "Nami external fee source is not supported"
	}
	// The two anti-sniper selectors only exist on the launch plugin.
	optional := client.NewRequest().SetContext(ctx).SetBlockNumber(block).SetOverrides(overrides)
	optional.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "antiSniperConfig"}, []any{&antiSniperOut})
	optional.AddCall(&ethrpc.Call{ABI: namiPluginABI, Target: target, Method: "startTimestamp"}, []any{&startTimestamp})
	resp, err := optional.TryAggregate()
	if err != nil {
		return nil, err
	}
	if resp.Result[0] != resp.Result[1] {
		return nil, errors.New("failed to read Nami anti-sniper state")
	}
	if resp.Result[0] {
		antiSniper := antiSniperOut.Config
		if antiSniper.StartingFee == nil {
			return nil, errors.New("failed to decode Nami anti-sniper state")
		}
		if antiSniper.StartingFee.Sign() != 0 && (startTimestamp == 0 || uint64(now) < startTimestamp+uint64(antiSniper.Period)) {
			unsupported = "Nami anti-sniper launch window is active"
		}
	}
	mevxGas, mevxUnsupported, err := mevxGasConfig(mevxRouter != (common.Address{}), minGasLeft, callGasBudget)
	if err != nil {
		return nil, err
	}
	if unsupported == "" {
		unsupported = mevxUnsupported
	}
	return &Config{Floor: bounds.Floor, Cap: bounds.Cap, Premium0For1: bounds.Premium0For1, Premium1For0: bounds.Premium1For0, DecayWindow: bounds.DecayWindow, MaxDiscountBps: bounds.MaxDiscountBps, Fee0For1: override.Fee0For1, Fee1For0: override.Fee1For0, ExpiresAt: override.ExpiresAt.Uint64(), MEVXGas: mevxGas, Unsupported: unsupported, MinGasLeft: minGasLeft.Uint64(), CallGasBudget: callGasBudget.Uint64()}, nil
}

// mevxGasConfig returns the per-hop MEV-X hook gas for an enabled router, or an
// unsupported reason when minGasLeft is too high to quote safely.
func mevxGasConfig(enabled bool, minGasLeft, callGasBudget *big.Int) (gas uint64, unsupported string, err error) {
	if !enabled {
		return 0, "", nil
	}
	if !minGasLeft.IsUint64() || !callGasBudget.IsUint64() || minGasLeft.Uint64() > 5000000 || callGasBudget.Uint64() > 5000000 {
		return 0, "", errors.New("invalid Nami MEV-X gas configuration")
	}
	if minGasLeft.Uint64() > MaxMinGasLeft {
		return 0, "Nami MEV-X minGasLeft is too high", nil
	}
	// callGasBudget only caps forwarding; it is not the amount consumed.
	return MEVXGasEstimate, "", nil
}

// HookGas is the gas added to a hop for the afterSwap hook: the typical hook
// cost plus the minGasLeft headroom the contract requires at that point.
// Snapshots stored before MinGasLeft existed carry zero and get the estimate only.
func (c Config) HookGas() uint64 {
	if c.MEVXGas == 0 {
		return 0
	}
	return MEVXGasEstimate + c.MinGasLeft
}

// fee follows _swapFee, after _feeForTick and with external fee sources disabled.
func (c Config) Fee(base uint16, zeroToOne bool, now uint64) uint16 {
	if c.Cap == 0 {
		return base
	}
	premium, override := c.Premium1For0, c.Fee1For0
	if zeroToOne {
		premium, override = c.Premium0For1, c.Fee0For1
	}
	side := uint64(base) + uint64(premium)
	fee := side
	if override != 0 {
		o := uint64(override)
		lowest := side - side*uint64(c.MaxDiscountBps)/10000
		if o < lowest {
			o = lowest
		}
		if o != side {
			if now < c.ExpiresAt {
				fee = o
			} else if now < c.ExpiresAt+uint64(c.DecayWindow) {
				remaining := c.ExpiresAt + uint64(c.DecayWindow) - now
				if o > side {
					fee = side + (o-side)*remaining/uint64(c.DecayWindow)
				} else {
					fee = side - (side-o)*remaining/uint64(c.DecayWindow)
				}
			}
		}
	}
	if fee < uint64(c.Floor) {
		fee = uint64(c.Floor)
	} else if fee > uint64(c.Cap) {
		fee = uint64(c.Cap)
	}
	return uint16(fee)
}
