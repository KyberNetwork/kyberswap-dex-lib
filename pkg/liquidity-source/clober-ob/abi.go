package cloberob

import (
	"bytes"
	_ "embed"

	abis "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/clober-ob/abi"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"
)

var (
	bookManagerABI abi.ABI
	bookViewerABI  abi.ABI
)

var (
	bookManagerFilterer *abis.BookManagerFilterer
)

func init() {
	builder := []struct {
		ABI  *abi.ABI
		data []byte
	}{
		{&bookManagerABI, bookManagerBytes},
		{&bookViewerABI, bookViewerBytes},
	}

	for _, b := range builder {
		var err error
		*b.ABI, err = abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
	}

	bookManagerFilterer = lo.Must(abis.NewBookManagerFilterer(common.Address{}, nil))
}
