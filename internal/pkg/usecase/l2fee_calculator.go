package usecase

import (
	"context"
	"errors"
	"math/big"
)

var (
	ErrChainIsNotL2 = errors.New("chain is not L2")
)

type L2FeeCalculatorUseCase struct {
	l2FeeCalculator  IL2FeeCalculator
	scannerStateRepo IScannerStateRepository
}

func NewL2FeeCalculatorUseCase(
	l2FeeCalculator IL2FeeCalculator,
	scannerStateRepo IScannerStateRepository,
) *L2FeeCalculatorUseCase {
	return &L2FeeCalculatorUseCase{
		l2FeeCalculator:  l2FeeCalculator,
		scannerStateRepo: scannerStateRepo,
	}
}

func (uc *L2FeeCalculatorUseCase) GetL1Fee(ctx context.Context, encodedSwapData string) (*big.Int, error) {
	if uc.l2FeeCalculator == nil {
		return nil, ErrChainIsNotL2
	}

	l2Fee, err := uc.scannerStateRepo.GetL2Fee(ctx)
	if err != nil {
		return nil, err
	}

	uc.l2FeeCalculator.SetParams(l2Fee)

	inputBytes, err := uc.l2FeeCalculator.CreateRawTxFromInputData(encodedSwapData)
	if err != nil {
		return nil, err
	}

	return uc.l2FeeCalculator.GetL1Fee(inputBytes), nil
}
