package repository

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository/gas"
)

type Config struct {
	Gas gas.Config `mapstructure:"gas"`
}
