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
		return nil, errors.New("Nami plugin is missing")
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
	var mevxGas uint64
	if mevxRouter != (common.Address{}) {
		if !minGasLeft.IsUint64() || !callGasBudget.IsUint64() || minGasLeft.Uint64() > 5000000 || callGasBudget.Uint64() > 5000000 {
			return nil, errors.New("invalid Nami MEV-X gas configuration")
		}
		// Account for the hook gas threshold, capped call, EIP-150 and hook overhead.
		mevxGas = minGasLeft.Uint64() + callGasBudget.Uint64()*64/63 + 100000
	}
	return &Config{Floor: bounds.Floor, Cap: bounds.Cap, Premium0For1: bounds.Premium0For1, Premium1For0: bounds.Premium1For0, DecayWindow: bounds.DecayWindow, MaxDiscountBps: bounds.MaxDiscountBps, Fee0For1: override.Fee0For1, Fee1For0: override.Fee1For0, ExpiresAt: override.ExpiresAt.Uint64(), MEVXGas: mevxGas, Unsupported: unsupported}, nil
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
