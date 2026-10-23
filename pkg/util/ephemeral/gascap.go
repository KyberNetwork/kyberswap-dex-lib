package ephemeral

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"
	"time"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
)

// Cap is the gas a node gives one eth_call.
type Cap struct {
	// Gas is the most gas one call runs with. 0 when Unlimited.
	Gas uint64
	// Unlimited: the node runs a call with whatever gas it is sent, and stops it only by time.
	Unlimited bool
}

// CallGasCeiling is the gas a call is sent when its caller leaves the gas to the node and the
// node allows more. A lens stops on gas, never on time: on a node without a gas cap, which stops a
// call only when its time runs out, a call sent unlimited gas would read a dense pool until that
// timeout, fail, and fail again on every retry. The densest pool measured, 11,272 ticks, took
// 140M gas in 1.6s; 100M keeps a call well inside geth's default 5s eth_call timeout, and pages
// such a pool in two calls.
const CallGasCeiling uint64 = 100_000_000

// probeCode is init code that returns the gas it starts with: GAS PUSH1 0 MSTORE PUSH1 32 PUSH1 0
// RETURN. Run as an eth_call without a to address, it costs a few gas.
var probeCode = []byte{0x5a, 0x60, 0x00, 0x52, 0x60, 0x20, 0x60, 0x00, 0xf3}

// probeIntrinsic is the gas a probe pays before its first opcode: 21000 for the call, 32000 for
// running init code, 112 + 8 for its 7 non-zero and 2 zero bytes, 2 for its one word of init code,
// and 2 for the GAS opcode itself.
const probeIntrinsic = 53_124

// probeLadder is the gas a probe asks for, highest first. A node caps a call above its limit, or,
// as Monad's does above 150M, refuses it; the first that runs tells the cap.
var probeLadder = []uint64{1 << 60, 1_000_000_000, 550_000_000, 150_000_000, 50_000_000, 30_000_000, 10_000_000}

// probeRetry is how long a failed probe is kept before a call probes again: a node that answers
// none of the ladder, or one rate limiting, is not asked again on every call.
const probeRetry = 5 * time.Minute

type probe struct {
	once sync.Once
	cap  Cap
	err  error
	at   time.Time
	// cut: the probe ended because its caller's context did, which says nothing of the node.
	cut bool
}

// gasCaps caches, per Caller, its node's Cap: a client's node does not change.
var gasCaps sync.Map

// GasCap returns the gas c's node gives one eth_call, probed once per client and kept. Nodes say
// it nowhere; the probe reads it off a call that returns the gas it was run with.
func GasCap(ctx context.Context, c Caller) (Cap, error) {
	if !reflect.TypeOf(c).Comparable() {
		return probeGasCap(ctx, c)
	}
	for {
		v, _ := gasCaps.LoadOrStore(c, &probe{})
		p := v.(*probe)
		p.once.Do(func() {
			p.cap, p.err = probeGasCap(ctx, c)
			p.cut = p.err != nil && ctx.Err() != nil
			p.at = time.Now()
		})
		if p.err != nil && time.Since(p.at) > probeRetry {
			gasCaps.CompareAndDelete(c, p)
			continue
		}
		if p.cut {
			// Cut short by its caller's context, the probe says nothing of the node: it is not kept,
			// and a caller whose context is still live probes again. A node's own timeout, which an
			// HTTP client reports as a deadline too, is kept as any other failure.
			gasCaps.CompareAndDelete(c, p)
			if err := ctx.Err(); err != nil {
				return Cap{}, err
			}
			continue
		}
		return p.cap, p.err
	}
}

func probeGasCap(ctx context.Context, c Caller) (Cap, error) {
	var errs []error
	for i, asked := range probeLadder {
		out, err := c.CallContract(ctx, ethereum.CallMsg{Data: probeCode, Gas: asked}, nil)
		if err == nil && len(out) != 32 {
			err = fmt.Errorf("probe returned %d bytes", len(out))
		}
		if err != nil {
			if ctx.Err() != nil {
				return Cap{}, ctx.Err()
			}
			errs = append(errs, fmt.Errorf("gas %d: %w", asked, err))
			continue
		}
		ran := new(big.Int).SetBytes(out).Uint64() + probeIntrinsic
		switch {
		case ran < asked-asked/100:
			// The node ran it with less than asked: its cap.
			return Cap{Gas: ran}, nil
		case i == 0:
			return Cap{Unlimited: true}, nil
		default:
			// Run whole after a higher ask was refused: the cap is at least this.
			return Cap{Gas: asked}, nil
		}
	}
	return Cap{}, fmt.Errorf("ephemeral: gas cap probe: %w", errors.Join(errs...))
}

// warnedCaps holds the clients already warned that a call's gas exceeds their cap.
var warnedCaps sync.Map

// callGas is the gas a call is sent: gas when the caller gives one, which is warned of once when it
// exceeds the node's cap; otherwise the node's cap up to CallGasCeiling, sent explicitly because a
// node's default can be far below its cap (Monad runs a call sent no gas with 8.1M, and up to 150M
// when asked). When the cap cannot be probed, 0 leaves it to the node.
func callGas(ctx context.Context, c Caller, gas uint64) uint64 {
	gasCap, err := GasCap(ctx, c)
	if gas != 0 {
		if err == nil && !gasCap.Unlimited && gas > gasCap.Gas {
			if _, warned := warnedCaps.LoadOrStore(c, true); !warned {
				logger.Warnf("ephemeral: lens gas %d exceeds the node's eth_call cap %d", gas, gasCap.Gas)
			}
		}
		return gas
	}
	switch {
	case err != nil:
		return 0
	case gasCap.Unlimited:
		return CallGasCeiling
	default:
		return min(gasCap.Gas, CallGasCeiling)
	}
}
