package erc20balanceslot

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/router-service/pkg/logger"
)

const (
	jsonRPCURL = "http://localhost:8545"
)

func TestWholeSlotWithFStrategy(t *testing.T) {
	t.Skip()

	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	rpcClient, err := rpc.Dial(jsonRPCURL)
	require.NoError(t, err)

	wallet := common.HexToAddress("0x19767032471665DF0FD7f6160381a103eCe6261A")
	p := NewWholeSlotWithFStrategy(rpcClient, wallet)

	token := common.HexToAddress("0x098754d293dd4375de7dC6566275aFb138021D00")
	bl, err := p.ProbeBalanceSlot(token, nil)
	require.NoError(t, err)
	require.True(t, bl.Found)
	spew.Dump(bl)
}

func TestMultipleStrategy(t *testing.T) {
	t.Skip()

	logger.InitLogger(logger.Configuration{
		EnableConsole: true,
		ConsoleLevel:  "debug",
	}, logger.LoggerBackendZap)

	rpcClient, err := rpc.Dial(jsonRPCURL)
	require.NoError(t, err)

	p := NewMultipleStrategy(rpcClient, randomizeAddress())

	token := common.HexToAddress("0x098754d293dd4375de7dC6566275aFb138021D00")
	bl, err := p.ProbeBalanceSlot(token, nil, &MultipleStrategyExtraParams{})
	require.NoError(t, err)
	require.True(t, bl.Found)
	require.NotEmpty(t, bl.StrategiesAttempted)
	spew.Dump(bl)
}
