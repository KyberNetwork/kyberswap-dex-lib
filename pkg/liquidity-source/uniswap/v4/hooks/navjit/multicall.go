package navjit

import (
	"context"
	"fmt"
	"maps"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
)

// cappedMulticallAddr is where the gas-capped multicall is injected. It is the canonical address of
// Uniswap's UniswapInterfaceMulticall, which holds no code on Robinhood.
var cappedMulticallAddr = common.HexToAddress("0x1F98415757620B543A52E61c46B32eB19261F984")

// cappedMulticallCode is UniswapInterfaceMulticall's runtime code, byte-identical to the deployment
// at the address above on Ethereum and Arbitrum One. multicall() runs each sub-call with
// call{gas: gasLimit}, so a reverting quote burns at most its own cap instead of 63/64 of what is
// left. It is not deployed on Robinhood, so each eth_call injects it with a state override.
var cappedMulticallCode = hexutil.MustDecode(cappedMulticallCodeHex)

// quoteGasCap is the gas limit of every sub-call (NAV, float and quotes) in a ladder multicall:
// 11/10 of the largest filling quote measured on the live venues, 2,913,996 (LOT-oldy buy $10; the
// others fill in 0.95-1.07M, see TestLive_QuoteGas). A rung that cannot fill burns the whole cap, so
// the longest round (12 USD rungs + 3 float rungs = 15 quotes, 48.1M) still fits the node's 50M cap.
const quoteGasCap = 3_205_396

type cappedCall struct {
	Target   common.Address
	GasLimit *big.Int
	CallData []byte
}

type cappedResult struct {
	Success    bool
	GasUsed    *big.Int
	ReturnData []byte
}

func newCappedCall(a abi.ABI, target common.Address, method string, params ...any) (cappedCall, error) {
	data, err := a.Pack(method, params...)
	return cappedCall{Target: target, GasLimit: new(big.Int).SetUint64(quoteGasCap), CallData: data}, err
}

// unpack decodes a successful sub-call's output of a's method into out; a failed or undecodable
// sub-call reports false.
func (r cappedResult) unpack(a abi.ABI, method string, out any) bool {
	return r.Success && a.UnpackIntoInterface(out, method, r.ReturnData) == nil
}

// cappedMulticall runs calls in one eth_call at block, each with a gas limit of gasCap. A sub-call
// that reverts or needs more than gasCap comes back with Success=false.
func cappedMulticall(ctx context.Context, rc *ethrpc.Client, overrides map[common.Address]gethclient.OverrideAccount,
	block *big.Int, calls []cappedCall) ([]cappedResult, error) {
	ov := make(map[common.Address]gethclient.OverrideAccount, len(overrides)+1)
	maps.Copy(ov, overrides)
	ov[cappedMulticallAddr] = gethclient.OverrideAccount{Code: cappedMulticallCode}

	var out struct {
		BlockNumber *big.Int
		ReturnData  []cappedResult
	}
	_, err := rc.NewRequest().SetContext(ctx).SetOverrides(ov).SetBlockNumber(block).
		AddCall(&ethrpc.Call{ABI: cappedMulticallABI, Target: hexutil.Encode(cappedMulticallAddr[:]),
			Method: "multicall", Params: []any{calls}}, []any{&out}).Call()
	if err != nil {
		return nil, err
	}
	if len(out.ReturnData) != len(calls) {
		return nil, fmt.Errorf("navjit: multicall returned %d results for %d calls", len(out.ReturnData), len(calls))
	}
	return out.ReturnData, nil
}
