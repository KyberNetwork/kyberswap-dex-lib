package gmxglp

import (
	"context"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"math/big"
	"strings"
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
	}, []interface{}{&glp})
	calls.AddCall(&ethrpc.Call{
		ABI:    glpManagerABI,
		Target: address,
		Method: glpManagerMethodGetAumInUsdg,
		Params: []interface{}{true},
	}, []interface{}{&maximiseAumInUsdg})
	calls.AddCall(&ethrpc.Call{
		ABI:    glpManagerABI,
		Target: address,
		Method: glpManagerMethodGetAumInUsdg,
		Params: []interface{}{false},
	}, []interface{}{&notMaximiseAumInUsdg})
	if _, err := calls.Aggregate(); err != nil {
		logger.Errorf("error when call aggreate request: %s", err)
		return nil, err
	}

	if _, err := g.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: glp.Hex(),
		Method: erc20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&glpTotalSupply}).Call(); err != nil {
		logger.Errorf("error when call request: %s", err)
		return nil, err
	}

	return &GlpManager{
		Glp:                  strings.ToLower(glp.Hex()),
		MaximiseAumInUsdg:    maximiseAumInUsdg,
		NotMaximiseAumInUsdg: notMaximiseAumInUsdg,
		GlpTotalSupply:       glpTotalSupply,
		StakeGlp:             g.config.StakeGLPAddress,
	}, nil
}
