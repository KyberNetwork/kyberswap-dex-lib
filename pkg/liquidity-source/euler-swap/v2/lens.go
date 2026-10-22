package v2

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/ephemeral"
)

//go:embed abis/lens.json
var lensJSON []byte

// lens is EulerSwapV2Lens, released by KyberNetwork/ephemeral-contracts.
var lens = ephemeral.MustLoad("EulerSwapV2Lens", lensJSON)

// poolInput is one pool for the lens. Vaults is the tracker's deduplicated vault list, in its
// order, with each vault's asset and quote amount (10^decimals).
type poolInput struct {
	Pool         common.Address
	EulerAccount common.Address
	Evc          common.Address
	Vaults       []common.Address
	VaultAssets  []common.Address
	QuoteAmounts []*big.Int
}

// callResult is one call's outcome, as a multicall's tryAggregate reports it.
type callResult struct {
	Success    bool
	ReturnData []byte
}

// poolReads is a pool's calls grouped by the tracker's four rounds, in the order the tracker
// builds them. A round the tracker skips for the pool is empty.
type poolReads struct {
	Round1 []callResult
	Round2 []callResult
	Round3 []callResult
	Round4 []callResult
}

// poolStates is what getPoolStates returns.
type poolStates struct {
	BlockNumber *big.Int
	Reads       []poolReads
}

// errLensReadsMismatch means the lens made a different number of calls in a round than the tracker
// builds for it: the two disagree about the pool, and the lens's reads cannot be trusted for it.
var errLensReadsMismatch = errors.New("euler-swap-v2: lens reads do not match the tracker's rounds")

// lensRead is one pool's four rounds as the lens read them, or why the lens could not read the
// pool.
type lensRead struct {
	reads poolReads
	err   error
}

// readLens reads every pool's four rounds through the lens, in as few calls as the node
// takes under gas (0 for the node's own cap), all at one block: block, or the latest when nil. The
// reads are in input order.
//
// A pool the lens cannot read even alone gets its error, and the rest are read without it: one
// pool whose read always reverts empty would otherwise take the whole batch with it.
func (d *PoolTracker) readLens(ctx context.Context, inputs []poolInput, block *big.Int,
	gas uint64) (uint64, []lensRead, error) {
	pin, err := ephemeral.Pin(ctx, d.ethrpcClient.GetETHClient(), block)
	if err != nil {
		return 0, nil, err
	}
	return ephemeral.Split(ctx, inputs, pin, lensTooLarge,
		func(ctx context.Context, part []poolInput, block *big.Int) (
			uint64, []lensRead, error) {
			var out poolStates
			err := ephemeral.Call(ctx, d.ethrpcClient.GetETHClient(), lens, &out, gas, block, part)
			if err != nil {
				if len(part) == 1 && lensTooLarge(err) {
					return block.Uint64(), []lensRead{{err: err}}, nil
				}
				return 0, nil, err
			}
			if !out.BlockNumber.IsUint64() {
				return 0, nil, fmt.Errorf("euler-swap-v2: lens read block %s", out.BlockNumber)
			}
			reads := make([]lensRead, len(out.Reads))
			for i, r := range out.Reads {
				reads[i].reads = r
			}
			return out.BlockNumber.Uint64(), reads, nil
		})
}

// lensTooLarge reports a call that needs fewer pools: the lens ran short of gas for its next read,
// or the call reverted with nothing, which is how nodes report running out of gas.
func lensTooLarge(err error) bool {
	return errors.Is(err, ephemeral.ErrNoRevertData) || ephemeral.IsContractError(err, "GasExhausted")
}

// lensInput is what the lens needs to make a pool's calls: the pool, its account and EVC, and the
// vault list getPoolData reads, deduplicated the way it deduplicates it.
func lensInput(poolAddress string, staticExtra StaticExtra, vaults []shared.VaultInfo) poolInput {
	input := poolInput{
		Pool:         common.HexToAddress(poolAddress),
		EulerAccount: common.HexToAddress(staticExtra.EulerAccount),
		Evc:          common.HexToAddress(staticExtra.EVC),
	}
	for _, v := range uniqueVaults(vaults) {
		input.Vaults = append(input.Vaults, common.HexToAddress(v.VaultAddress))
		input.VaultAssets = append(input.VaultAssets, common.HexToAddress(v.AssetAddress))
		input.QuoteAmounts = append(input.QuoteAmounts, v.QuoteAmount)
	}
	return input
}

// lensReplay feeds a pool's recorded lens results to getPoolData in place of its multicalls.
type lensReplay struct {
	block  *big.Int
	rounds [4][]callResult
	used   [4]bool
}

func newLensReplay(block *big.Int, reads poolReads) *lensReplay {
	return &lensReplay{block: block, rounds: [4][]callResult{reads.Round1, reads.Round2,
		reads.Round3, reads.Round4}}
}

// exec decodes a round's recorded results into its calls' outputs the way ethrpc decodes a
// tryAggregate: a failed call leaves its outputs untouched, and a successful one is unpacked with
// the first of its ABIs that fits.
func (r *lensReplay) exec(req *ethrpc.Request, round int) (*big.Int, error) {
	results := r.rounds[round]
	if len(results) != len(req.Calls) {
		return nil, fmt.Errorf("%w: round %d has %d results for %d calls", errLensReadsMismatch, round+1,
			len(results), len(req.Calls))
	}
	r.used[round] = true

	for i, c := range req.Calls {
		if !results[i].Success {
			continue
		}
		for j, unpackABI := range c.UnpackABI {
			if err := unpackABI.UnpackIntoInterface(c.Output[j], c.Method, results[i].ReturnData); err == nil {
				break
			}
		}
	}

	if round <= 1 {
		return r.block, nil
	}
	return nil, nil
}

// consumed fails when the lens recorded calls for a round the tracker never ran.
func (r *lensReplay) consumed() error {
	for i, used := range r.used {
		if !used && len(r.rounds[i]) > 0 {
			return fmt.Errorf("%w: round %d has %d results the tracker did not ask for", errLensReadsMismatch, i+1,
				len(r.rounds[i]))
		}
	}
	return nil
}
