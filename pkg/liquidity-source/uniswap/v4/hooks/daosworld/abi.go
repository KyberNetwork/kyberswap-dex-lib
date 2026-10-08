package daosworld

import (
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/samber/lo"
)

var hookABI = lo.Must(abi.JSON(strings.NewReader(
	// language=json
	`[{"type":"function","name":"launches","stateMutability":"view","inputs":[{"name":"token","type":"address"}],
"outputs":[{"name":"feeRecipient","type":"address"},{"name":"startFeeBips","type":"uint16"},
{"name":"endFeeBips","type":"uint16"},{"name":"decayDuration","type":"uint32"},{"name":"launchTime","type":"uint32"}]}]`,
)))
