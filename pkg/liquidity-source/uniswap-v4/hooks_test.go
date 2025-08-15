package uniswapv4

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestHasSwapPermissions(t *testing.T) {
	hook := "0xd73339564ac99f3e09b0ebc80603ff8b796500c0"
	t.Log(HasSwapPermissions(common.HexToAddress(hook)))
}
