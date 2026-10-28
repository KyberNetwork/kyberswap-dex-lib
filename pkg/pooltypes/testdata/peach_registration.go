package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common"

	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	_ "github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
)

func main() {
	for _, a := range []string{"0x4D1546682BC8252E52c1c1820F0AaA7694C9e080", "0x1e4605837970EeF394662B1a9e88e09292636080", "0x98C819a522b91f76bAEAf499F3801e76890F2080", "0x8c960De62a26A688B72e6cbf644F46715642a080", "0x3722d2840bFA4E9610d265DA08078d178bF46080"} {
		if _, ok := v4.HookFactories[common.HexToAddress(a)]; !ok {
			fmt.Fprintln(os.Stderr, "Peach factory missing:", a)
			os.Exit(1)
		}
	}
}
