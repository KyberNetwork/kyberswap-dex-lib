package tokentax

import (
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/holiman/uint256"
)

func CallSucceeded(response *ethrpc.Response, index int) bool {
	return response != nil &&
		index >= 0 &&
		index < len(response.Result) &&
		response.Result[index]
}

func ToUint256(value *big.Int) *uint256.Int {
	if value == nil {
		return nil
	}
	result, _ := uint256.FromBig(value)
	return result
}

func FindPairedToken(pool entity.Pool, baseTokens map[string]struct{}) string {
	if len(pool.Tokens) != 2 {
		return ""
	}
	for i, token := range pool.Tokens {
		if _, ok := baseTokens[strings.ToLower(token.Address)]; ok {
			return strings.ToLower(pool.Tokens[1-i].Address)
		}
	}
	return ""
}
