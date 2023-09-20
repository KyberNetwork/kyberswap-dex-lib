package swapdata

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

// Uniswap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper1.sol#L22-L31
type Uniswap struct {
	Pool             common.Address
	TokenIn          common.Address
	TokenOut         common.Address
	Recipient        common.Address
	CollectAmount    *big.Int
	SwapFee          uint32
	FeePrecision     uint32
	TokenWeightInput uint32
}

// StableSwap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L41-L51
type StableSwap struct {
	Pool           common.Address
	TokenFrom      common.Address
	TokenTo        common.Address
	TokenIndexFrom uint8
	TokenIndexTo   uint8
	Dx             *big.Int
	PoolLength     *big.Int
	PoolLp         common.Address
	IsSaddle       bool
}

// CurveSwap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L53-L62
type CurveSwap struct {
	Pool              common.Address
	TokenFrom         common.Address
	TokenTo           common.Address
	TokenIndexFrom    *big.Int
	TokenIndexTo      *big.Int
	Dx                *big.Int
	UsePoolUnderlying bool
	UseTriCrypto      bool
}

// UniswapV3KSElastic
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L64-L72
type UniswapV3KSElastic struct {
	Recipient         common.Address
	Pool              common.Address
	TokenIn           common.Address
	TokenOut          common.Address
	SwapAmount        *big.Int
	SqrtPriceLimitX96 *big.Int
	IsUniV3           bool
}

// BalancerV2
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L83-L89
type BalancerV2 struct {
	Vault    common.Address
	PoolId   [32]byte
	AssetIn  common.Address
	AssetOut common.Address
	Amount   *big.Int
}

// DODO
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L99-L108
type DODO struct {
	Recipient  common.Address
	Pool       common.Address
	TokenFrom  common.Address
	TokenTo    common.Address
	Amount     *big.Int
	SellHelper common.Address
	IsSellBase bool
	IsVersion2 bool
}

// GMX
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L110-L116
type GMX struct {
	Vault    common.Address
	TokenIn  common.Address
	TokenOut common.Address
	Amount   *big.Int
	Receiver common.Address
}

// Synthetix
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/921725af2a121e023945fa46669c3ea5343ecd37/contracts/executor-helpers/ExecutorHelper2.sol#L118-L126
type Synthetix struct {
	SynthetixProxy         common.Address
	TokenIn                common.Address
	TokenOut               common.Address
	SourceCurrencyKey      [32]byte
	SourceAmount           *big.Int
	DestinationCurrencyKey [32]byte
	UseAtomicExchange      bool
}

// PSM
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/35d5ffa3388f058055d5bf99eeef943daad348f8/contracts/executor-helpers/ethereum/ExecutorHelperEthereum1.sol#L135-L141
type PSM struct {
	Router    common.Address
	TokenIn   common.Address
	TokenOut  common.Address
	AmountIn  *big.Int
	Recipient common.Address
}

// WSTETH
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/35d5ffa3388f058055d5bf99eeef943daad348f8/contracts/executor-helpers/ethereum/ExecutorHelperEthereum1.sol#L148-L152
type WSTETH struct {
	Pool       common.Address
	Amount     *big.Int
	IsWrapping bool
}

type Platypus struct {
	Pool              common.Address
	TokenIn           common.Address
	TokenOut          common.Address
	Recipient         common.Address
	CollectAmount     *big.Int
	LimitReturnAmount *big.Int
}

// Reference from SC code
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/edd5870ecd990313cb9ab984b7d6a4f16ad6ed9b/contracts/interfaces/IKyberLimitOrder.sol#L5
type Order struct {
	Salt                 *big.Int
	MakerAsset           common.Address
	TakerAsset           common.Address
	Maker                common.Address
	Receiver             common.Address
	AllowedSender        common.Address
	MakingAmount         *big.Int
	TakingAmount         *big.Int
	FeeRecipient         common.Address
	MakerTokenFeePercent uint32
	MakerAssetData       []byte
	TakerAssetData       []byte
	GetMakerAmount       []byte
	GetTakerAmount       []byte
	Predicate            []byte
	Permit               []byte
	Interaction          []byte
}

// Reference from SC code
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/edd5870ecd990313cb9ab984b7d6a4f16ad6ed9b/contracts/interfaces/IKyberLimitOrder.sol#L23
type FillBatchOrdersParams struct {
	Orders          []Order
	Signatures      [][]byte
	TakingAmount    *big.Int
	ThresholdAmount *big.Int
	Target          common.Address
}

// Reference from SC code
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/edd5870ecd990313cb9ab984b7d6a4f16ad6ed9b/contracts/executor-helpers/ExecutorHelper1.sol#L147
type KyberLimitOrder struct {
	KyberLOAddress common.Address
	MakerAsset     common.Address
	TakerAsset     common.Address
	Params         FillBatchOrdersParams
}

// Double Signature Limit Order
type OrderDS struct {
	Salt           *big.Int
	MakerAsset     common.Address
	TakerAsset     common.Address
	Maker          common.Address
	Receiver       common.Address
	AllowedSender  common.Address
	MakingAmount   *big.Int
	TakingAmount   *big.Int
	FeeConfig      *big.Int
	MakerAssetData []byte
	TakerAssetData []byte
	GetMakerAmount []byte
	GetTakerAmount []byte
	Predicate      []byte
	Permit         []byte
	Interaction    []byte
}

type Signature struct {
	OrderSignature []byte // Signature to confirm quote ownership
	OpSignature    []byte // OP Signature to confirm quote ownership
}

type FillBatchOrdersParamsDS struct {
	Orders          []OrderDS
	Signatures      []Signature
	OpExpireTimes   []uint32
	TakingAmount    *big.Int
	ThresholdAmount *big.Int
	Target          common.Address
}

type KyberLimitOrderDS struct {
	KyberLOAddress common.Address
	MakerAsset     common.Address
	TakerAsset     common.Address
	Params         FillBatchOrdersParamsDS
}

// SyncSwap
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop_zk/contracts/executor-helpers/ZkSyncExecutorHelper.sol#L72-L77
type SyncSwap struct {
	Data          []byte
	Vault         common.Address
	TokenIn       common.Address
	Pool          common.Address
	CollectAmount *big.Int
}

type MaverickV1Swap struct {
	Pool              common.Address
	TokenIn           common.Address
	TokenOut          common.Address
	Recipient         common.Address
	SwapAmount        *big.Int
	SqrtPriceLimitD18 *big.Int
}

type AlgebraV1 struct {
	Recipient           common.Address
	Pool                common.Address
	TokenIn             common.Address
	TokenOut            common.Address
	SwapAmount          *big.Int
	SqrtPriceLimitX96   *big.Int
	SenderFeeOnTransfer *big.Int
}

type TraderJoeV2 struct {
	Recipient common.Address
	Pool      common.Address
	TokenIn   common.Address
	TokenOut  common.Address
	// packed (version, collectAmount)
	PackedCollectAmount *big.Int `abi:"collectAmount"`
}

// OrderRFQ
// Reference: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/fda542505b49252f6c59273d9ee542377be6c3a9/contracts/interfaces/pool-types/IRFQ.sol#L7-L18
type OrderRFQ struct {
	// lowest 64 bits is the order id, next 64 bits is the expiration timestamp
	// highest bit is unwrap WETH flag which is set on taker's side
	// [unwrap eth(1 bit) | unused (127 bits) | expiration timestamp(64 bits) | orderId (64 bits)]
	Info          *big.Int
	MakerAsset    common.Address
	TakerAsset    common.Address
	Maker         common.Address
	AllowedSender common.Address // null address on public orders
	MakingAmount  *big.Int
	TakingAmount  *big.Int
}

// KyberRFQ
// Reference: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/fda542505b49252f6c59273d9ee542377be6c3a9/contracts/executor-helpers/ExecutorHelper2.sol#L91-L97
type KyberRFQ struct {
	RFQ       common.Address `abi:"rfq"`
	Order     OrderRFQ
	Signature []byte
	Amount    *big.Int
	Target    common.Address
}

// IZiSwap
// Reference: https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/29d63d83cac3011067bcfd3b9597239745f848d9/contracts/executor-helpers/ExecutorHelper3.sol#L77C1-L84C4
type IZiSwap struct {
	Pool       common.Address
	TokenIn    common.Address
	TokenOut   common.Address
	Recipient  common.Address
	SwapAmount *big.Int
	LimitPoint *big.Int
}

// Wombat
// https://github.com/KyberNetwork/ks-dex-aggregator-sc/blob/develop/contracts/executor-helpers/ExecutorHelper3.sol#L335
type Wombat struct {
	Pool      common.Address
	TokenIn   common.Address
	TokenOut  common.Address
	Amount    *big.Int
	Recipient common.Address
}
