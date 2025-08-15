package helper

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/decode"
	"github.com/ethereum/go-ethereum/common"
)

type Interaction struct {
	Target common.Address
	Data   []byte
}

func (i Interaction) IsZero() bool {
	return i.Target.String() == common.Address{}.String() && len(i.Data) == 0
}

func (i Interaction) Encode() []byte {
	res := make([]byte, 0, len(i.Target)+len(i.Data))
	return append(append(res, i.Target.Bytes()...), i.Data...)
}

func DecodeInteraction(data []byte) (Interaction, error) {
	bi := decode.NewBytesIterator(data)
	target, err := bi.NextBytes(common.AddressLength)
	if err != nil {
		return Interaction{}, fmt.Errorf("get target: %w", err)
	}
	return Interaction{
		Target: common.BytesToAddress(target),
		Data:   bi.RemainingData(),
	}, nil
}
