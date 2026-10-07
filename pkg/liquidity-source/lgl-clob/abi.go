package lglclob

import (
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"

	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

var (
	onchainClobABI = abiutil.MustParseABI(OnchainClobABIJson)
	lensABI        = abiutil.MustParseABI(lensABIJson)
	lensBytecode   = hexutil.MustDecode(strings.TrimSpace(lensBytecodeHex))
)
