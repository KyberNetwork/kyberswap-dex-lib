package shared

const (
	ZeroPoolID               = "0x0000000000000000000000000000000000000000000000000000000000000000"
	VaultMethodGetPoolTokens = "getPoolTokens"

	JoinExitGasUsage int64 = 150000
)

type JoinExitKind int64

const (
	PoolExit JoinExitKind = 0
	PoolJoin JoinExitKind = 1
)
