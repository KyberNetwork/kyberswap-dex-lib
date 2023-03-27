package dodo

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSellBaseROne(t *testing.T) {
	amountIn := big.NewFloat(1)
	var poolState = PoolState{
		B:           big.NewFloat(12046601.584215026),
		Q:           big.NewFloat(15353622.282616144),
		B0:          big.NewFloat(6023300.792107513087938603),
		Q0:          big.NewFloat(7676811.141308072430478088),
		RStatus:     rStatusOne,
		OraclePrice: big.NewFloat(1),
		k:           big.NewFloat(0.0002),
		mtFeeRate:   big.NewFloat(0.00002),
		lpFeeRate:   big.NewFloat(0),
	}
	amountOut, err := QuerySellBase(amountIn, &poolState)

	assert.Equal(t, err, nil)
	assert.Equal(t, amountOut.String(), "0.9999799991")
}

func TestSellBaseRAboveOne(t *testing.T) {
	amountIn := big.NewFloat(1)
	var poolState = PoolState{
		B:           big.NewFloat(2208481.244464409851881798),
		Q:           big.NewFloat(11492948.594477208115740703),
		B0:          big.NewFloat(6023300.792107513087938603),
		Q0:          big.NewFloat(7676811.141308072430478088),
		RStatus:     rStatusAboveOne,
		OraclePrice: big.NewFloat(1),
		k:           big.NewFloat(0.0002),
		mtFeeRate:   big.NewFloat(0.00002),
		lpFeeRate:   big.NewFloat(0),
	}
	amountOut, err := QuerySellBase(amountIn, &poolState)

	assert.Equal(t, err, nil)
	assert.Equal(t, amountOut.String(), "1.001267661")
}

func TestSellBaseRBelowOne(t *testing.T) {
	amountIn := big.NewFloat(1)
	var poolState = PoolState{
		B:           big.NewFloat(2208481.244464409851881798),
		Q:           big.NewFloat(11492948.594477208115740703),
		B0:          big.NewFloat(6023300.792107513087938603),
		Q0:          big.NewFloat(7676811.141308072430478088),
		RStatus:     rStatusBelowOne,
		OraclePrice: big.NewFloat(1),
		k:           big.NewFloat(0.0002),
		mtFeeRate:   big.NewFloat(0.00002),
		lpFeeRate:   big.NewFloat(0),
	}
	amountOut, err := QuerySellBase(amountIn, &poolState)

	assert.Equal(t, err, nil)
	assert.Equal(t, amountOut.String(), "1.000090777")
}

func TestSellQuoteROne(t *testing.T) {
	amountIn := big.NewFloat(1)
	var poolState = PoolState{
		B:           big.NewFloat(12046601.584215026),
		Q:           big.NewFloat(15353622.282616144),
		B0:          big.NewFloat(6023300.792107513087938603),
		Q0:          big.NewFloat(7676811.141308072430478088),
		RStatus:     rStatusOne,
		OraclePrice: big.NewFloat(1),
		k:           big.NewFloat(0.0002),
		mtFeeRate:   big.NewFloat(0.00002),
		lpFeeRate:   big.NewFloat(0),
	}
	amountOut, err := QuerySellQuote(amountIn, &poolState)

	assert.Equal(t, err, nil)
	assert.Equal(t, amountOut.String(), "0.99998")
}

func TestSellQuoteRAboveOne(t *testing.T) {
	amountIn := big.NewFloat(1)
	var poolState = PoolState{
		B:           big.NewFloat(2208481.244464409851881798),
		Q:           big.NewFloat(11492948.594477208115740703),
		B0:          big.NewFloat(6023300.792107513087938603),
		Q0:          big.NewFloat(7676811.141308072430478088),
		RStatus:     rStatusAboveOne,
		OraclePrice: big.NewFloat(1),
		k:           big.NewFloat(0.0002),
		mtFeeRate:   big.NewFloat(0.00002),
		lpFeeRate:   big.NewFloat(0),
	}
	amountOut, err := QuerySellQuote(amountIn, &poolState)

	assert.Equal(t, err, nil)
	assert.Equal(t, amountOut.String(), "0.9986939936")
}
