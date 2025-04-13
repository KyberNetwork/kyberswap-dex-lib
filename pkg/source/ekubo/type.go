package ekubo

import (
	"errors"
	"fmt"
	"math/big"
	"slices"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type Extra struct {
	State quoting.PoolState `json:"state"`
}

type StaticExtra struct {
	PoolKey   quoting.PoolKey `json:"poolKey"`
	Extension pool.Extension  `json:"extension"`
}

type addressWrapper struct {
	common.Address
}

func (b *addressWrapper) UnmarshalJSON(input []byte) error {
	if len(input) <= 4 {
		return errors.New("expected non-empty prefixed hex string")
	}

	hexString := input[1 : len(input)-1]
	if len(hexString)%2 != 0 {
		hexString = slices.Insert(hexString, 3, []byte("0")[0])
	}
	bytes, err := hexutil.Decode(string(hexString))
	if err != nil {
		return fmt.Errorf("decoding hex string: %w", err)
	}

	b.Address = common.BytesToAddress(bytes)

	return nil
}

type uint64UncheckedWrapper struct {
	uint64
}

// TODO Make checked once we use another endpoint that doesn't include 0.128 fees
func (b *uint64UncheckedWrapper) UnmarshalJSON(input []byte) error {
	if len(input) <= 4 {
		return errors.New("expected non-empty prefixed hex string")
	}

	bi := new(big.Int)
	if err := bi.UnmarshalJSON(input[1 : len(input)-1]); err != nil {
		return fmt.Errorf("parsing big int: %w", err)
	}

	b.uint64 = bi.Uint64()

	return nil
}
