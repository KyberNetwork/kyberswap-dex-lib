package service

import (
	"context"
)

type IService interface {
	UpdateData(ctx context.Context)
}
