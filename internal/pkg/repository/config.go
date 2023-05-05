package repository

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
)

type Config struct {
	Gas      gas.Config      `mapstructure:"gas"`
	Pool     pool.Config     `mapstructure:"pool"`
	Price    price.Config    `mapstructure:"price"`
	Token    token.Config    `mapstructure:"token"`
	PoolRank poolrank.Config `mapstructure:"poolRank"`
	Route    route.Config    `mapstructure:"route"`
}
