package executor

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/ethereum/go-ethereum/common"
)

type SwapExecutorDescription struct {
	SwapSequences        []byte
	TokenIn              common.Address
	TokenOut             common.Address
	To                   common.Address
	Deadline             *big.Int
	PositiveSlippageData []byte
}

type PositiveSlippageFeeData struct {
	PartnerReceiver      pack.UInt160
	PartnerPercent       pack.UInt96
	ExpectedReturnAmount *big.Int
	MinimumPSAmount      *big.Int
}
