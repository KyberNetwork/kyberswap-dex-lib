package executor

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// CallBytesInputs inputs of executor contract callBytes function
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/35d5ffa3388f058055d5bf99eeef943daad348f8/contracts/AggregationExecutor.sol#L61
type CallBytesInputs struct {
	Data SwapExecutorDescription
}

// SimpleSwapData contains data for simple swap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/35d5ffa3388f058055d5bf99eeef943daad348f8/contracts/MetaAggregationRouterV2.sol#L58-L64
type SimpleSwapData struct {
	// FirstPools addresses of first pools of each swap sequence
	FirstPools []common.Address

	// FirstSwapAmounts amount of token to be swapped in first swap of each swap sequence
	FirstSwapAmounts []*big.Int

	// SwapDatas array of packed SwapSequence
	SwapDatas [][]byte

	// Deadline swap deadline
	Deadline *big.Int

	// DestTokenFeeData is packed fee data
	DestTokenFeeData []byte
}

// SwapExecutorDescription contains data required by executor contract to execute swap in normal mode
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/interfaces/IAggregationExecutor.sol#L29-L36
type SwapExecutorDescription struct {
	// SwapSequences contains detail Swap
	SwapSequences [][]Swap

	// TokenIn address of the token to be swapped
	TokenIn common.Address

	// TokenOut address of the token to be received
	TokenOut common.Address

	// To address of wallet that token out will be transferred to
	To common.Address

	// Deadline swap deadline
	Deadline *big.Int

	// DestTokenFeeData is packed fee data
	DestTokenFeeData []byte
}

// Swap contains data of a swap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/interfaces/IAggregationExecutor.sol#L8-L11
type Swap struct {
	// Data is packed swap data
	Data []byte

	// SelectorAndFlags [selector (32 bits) + flags (224 bits)]
	SelectorAndFlags SwapSelectorAndFlags
}
type SwapSelectorAndFlags [32]byte
type SwapSelector [4]byte
type SwapFlags [4]byte

// SwapSingleSequenceInputs inputs of swapSingleSequence function
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/35d5ffa3388f058055d5bf99eeef943daad348f8/contracts/AggregationExecutor.sol#L130
type SwapSingleSequenceInputs struct {
	SwapData []Swap
}

type PositiveSlippageFeeData struct {
	ExpectedReturnAmount *big.Int
}
