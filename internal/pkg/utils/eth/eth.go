package eth

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func IsEther(tokenAddress string) bool {
	return strings.EqualFold(tokenAddress, valueobject.EtherAddress)
}

func IsWETH(tokenAddress string, chainID valueobject.ChainID) bool {
	return strings.EqualFold(tokenAddress, valueobject.WETHByChainID[chainID])
}

// ConvertEtherToWETH converts token to WETH if token is equal to ether
func ConvertEtherToWETH(tokenAddress string, chainID valueobject.ChainID) (string, error) {
	if !IsEther(tokenAddress) {
		return tokenAddress, nil
	}

	weth, ok := valueobject.WETHByChainID[chainID]
	if !ok {
		return tokenAddress, errors.Wrapf(
			ErrWETHNotFound,
			"chainID: [%v]",
			chainID,
		)
	}

	return strings.ToLower(weth), nil
}

func ConvertWETHToEther(tokenAddress string, chainID valueobject.ChainID) (string, error) {
	wethAddress, ok := valueobject.WETHByChainID[chainID]
	if !ok {
		return tokenAddress, errors.Wrapf(
			ErrWETHNotFound,
			"chainID: [%v]",
			chainID,
		)
	}

	if strings.EqualFold(tokenAddress, wethAddress) {
		return strings.ToLower(valueobject.EtherAddress), nil
	}

	return tokenAddress, nil
}
