package prop

import (
	"bytes"
	"context"
	"math/big"
	"slices"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

// snapshot mirrors KipseliPropLens.Snapshot field-for-field (same order and
// types), so the unpacked tuple converts into it directly.
type snapshot struct {
	BlockNumber    *big.Int
	BlockTimestamp *big.Int
	SwapImpl       common.Address
	Wallet         common.Address
	QuoteToken     common.Address
	Quoter         common.Address
	ListedTokens   []common.Address
	Balances       []*big.Int
	Caps           []*big.Int
	AmountsOut     [][]*big.Int
}

// fetchSnapshot runs KipseliPropLens in one `to`-less eth_call. Pass no tokens
// to only list. titanState's overrides, when present, apply to the whole call
// at the block timestamp Titan pushed them for.
func fetchSnapshot(
	ctx context.Context, client *ethrpc.Client, router, dest common.Address,
	tokens []common.Address, amountsIn [][]*big.Int, titanState titan.State,
) (*snapshot, error) {
	if tokens == nil {
		tokens = []common.Address{}
	}
	if amountsIn == nil {
		amountsIn = [][]*big.Int{}
	}
	args, err := lensABI.Pack("", router, dest, tokens, amountsIn)
	if err != nil {
		return nil, err
	}

	// Simulate the block Titan built the state for, number included: oracles
	// keyed by block.number (e.g. fermi's) panic or report stale on head.
	var blockOverrides *gethclient.BlockOverrides
	if titanState.BlockTimestamp != 0 {
		blockOverrides = &gethclient.BlockOverrides{Number: titanState.BlockNumber, Time: titanState.BlockTimestamp}
	}
	data, err := eth.DeploylessCall(ctx, client.GetETHClient().Client(),
		slices.Concat(lensBytecode, args), titanState.Overrides, blockOverrides)
	if err != nil {
		return nil, err
	}
	return decodeSnapshot(data)
}

func decodeSnapshot(data []byte) (*snapshot, error) {
	lensErr := lensABI.Errors[lensSnapshotError]
	if len(data) < 4 || !bytes.Equal(data[:4], lensErr.ID[:4]) {
		return nil, ErrUnexpectedLensRevert
	}
	values, err := lensErr.Inputs.Unpack(data[4:])
	if err != nil {
		return nil, err
	}
	return abi.ConvertType(values[0], new(snapshot)).(*snapshot), nil
}
