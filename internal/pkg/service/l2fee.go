package service

import (
	"context"
	"time"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/timer"
)

const (
	updateL2FeeInterval = 30 * time.Second
)

type (
	IScannerStateRepository interface {
		SetL2Fee(ctx context.Context, l2fee *entity.L2Fee) error
	}

	IL2FeeReader interface {
		Read(ctx context.Context) (*entity.L2Fee, error)
	}
)

type L2FeeService struct {
	reader           IL2FeeReader
	scannerStateRepo IScannerStateRepository
}

func NewL2FeeService(
	reader IL2FeeReader,
	scannerStateRepo IScannerStateRepository,
) *L2FeeService {
	return &L2FeeService{
		reader:           reader,
		scannerStateRepo: scannerStateRepo,
	}
}

func (s *L2FeeService) UpdateData(ctx context.Context) {
	run := func() error {
		finish := timer.Start("updating L2 fee")
		defer finish()

		l2fee, err := s.reader.Read(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to query L2 fee")
		}

		if err := s.scannerStateRepo.SetL2Fee(ctx, l2fee); err != nil {
			return errors.Wrap(err, "failed to update L2 fee")
		}

		return nil
	}

	for {
		err := run()
		if err != nil {
			logger.Errorf("failed to update L2 fee, err: %v", err)
		}

		time.Sleep(updateL2FeeInterval)
	}
}
