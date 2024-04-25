package repository

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository/blackjack"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/erc20balanceslot"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
	onchainprice "github.com/KyberNetwork/router-service/internal/pkg/repository/onchain-price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pool"
	poolservice "github.com/KyberNetwork/router-service/internal/pkg/repository/pool-service"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/poolrank"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/price"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/route"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/token"
)

type Config struct {
	Gas              gas.Config              `json:"gas" mapstructure:"gas"`
	Pool             pool.Config             `json:"pool" mapstructure:"pool"`
	Price            price.Config            `json:"price" mapstructure:"price"`
	OnchainPrice     onchainprice.Config     `json:"onchainprice" mapstructure:"onchainprice"`
	Token            token.Config            `json:"token" mapstructure:"token"`
	PoolRank         poolrank.Config         `json:"poolRank" mapstructure:"poolRank"`
	Route            route.Config            `json:"route" mapstructure:"route"`
	Blackjack        blackjack.Config        `json:"blackjack" mapstructure:"blackjack"`
	ERC20BalanceSlot erc20balanceslot.Config `json:"erc20balanceslot" mapstructure:"erc20balanceslot"`
	PoolService      poolservice.Config      `json:"poolService" mapstructure:"poolService"`
}
