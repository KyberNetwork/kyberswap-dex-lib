package gmxglp

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type GlpManagerScanner struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewGlpManagerScanner(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *GlpManagerScanner {
	return &GlpManagerScanner{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (g *GlpManagerScanner) getGlpManager(ctx context.Context, address string) (*GlpManager, error) {
	var glp common.Address
	var glpTotalSupply, maximiseAumInUsdg, notMaximiseAumInUsdg *big.Int

	calls := g.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    glpManagerABI,
		Target: address,
		Method: glpManagerMethodGlp,
		Params: nil,
	}, []any{&glp})
	calls.AddCall(&ethrpc.Call{
		ABI:    glpManagerABI,
		Target: address,
		Method: glpManagerMethodGetAumInUsdg,
		Params: []any{true},
	}, []any{&maximiseAumInUsdg})
	calls.AddCall(&ethrpc.Call{
		ABI:    glpManagerABI,
		Target: address,
		Method: glpManagerMethodGetAumInUsdg,
		Params: []any{false},
	}, []any{&notMaximiseAumInUsdg})
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("error when call aggreate request: %s", err)
		return nil, err
	}

	if _, err := g.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: glp.Hex(),
		Method: erc20MethodTotalSupply,
		Params: nil,
	}, []any{&glpTotalSupply}).Call(); err != nil {
		logger.Errorf("error when call request: %s", err)
		return nil, err
	}

	return &GlpManager{
		Address:              address,
		Glp:                  hexutil.Encode(glp[:]),
		MaximiseAumInUsdg:    maximiseAumInUsdg,
		NotMaximiseAumInUsdg: notMaximiseAumInUsdg,
		GlpTotalSupply:       glpTotalSupply,
		StakeGlp:             g.config.StakeGLPAddress,
	}, nil
}
