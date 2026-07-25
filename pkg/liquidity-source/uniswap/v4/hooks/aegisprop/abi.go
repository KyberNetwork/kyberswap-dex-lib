package aegisprop

import (
	"bytes"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var aegisPropHookABI = lo.Must(abi.JSON(bytes.NewReader(aegisPropHookABIJson)))
