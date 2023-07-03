package curveAave

import "github.com/KyberNetwork/router-service/internal/pkg/utils"

var (
	DefaultGas = Gas{ExchangeUnderlying: 495000}
	Precision  = utils.NewBig10("1000000000000000000")
)
