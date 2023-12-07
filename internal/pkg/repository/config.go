package repository

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository/blackjack"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
)

type Config struct {
	Gas       gas.Config       `json:"gas" mapstructure:"gas"`
	Pool      pool.Config      `json:"pool" mapstructure:"pool"`
	Price     price.Config     `json:"price" mapstructure:"price"`
	Token     token.Config     `json:"token" mapstructure:"token"`
	PoolRank  poolrank.Config  `json:"poolRank" mapstructure:"poolRank"`
	Route     route.Config     `json:"route" mapstructure:"route"`
	Blackjack blackjack.Config `json:"blackjack" mapstructure:"blackjack"`
}
