package service

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/config"
	"github.com/KyberNetwork/kyberswap-aggregator/pkg/redis"
)

type IService interface {
	UpdateData(ctx context.Context)
}

type App struct {
	DB     *gorm.DB
	Redis  redis.DataStoreRepository
	Router *gin.RouterGroup
	Config *config.Common

	RPCService   *RPCService
	TokenService *TokenService
	PriceService *ScanService
}
