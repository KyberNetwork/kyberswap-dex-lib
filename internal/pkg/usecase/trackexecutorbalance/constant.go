package trackexecutor

import (
	"time"
)

const (
	graphQLRequestTimeout = 20 * time.Second
	graphQLPageSize       = 1000
	graphQLMaxOffset      = 5000

	erc20MethodGetAllowance = "allowance"
	erc20MethodGetBalanceOf = "balanceOf"

	EtherAddress = "0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"

	intervalDelay = 5 * time.Second
)
