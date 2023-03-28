package limitorder

import (
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

const (
	defaultUpdateReserveBulk  = 10
	defaultReserveJobInterval = time.Duration(10 * time.Second)
)

type (
	Extra struct {
		SellOrders []*valueobject.Order
		BuyOrders  []*valueobject.Order
	}

	Properties struct {
		LimitOrderHTTPUrl       string                   `json:"limitOrderHTTPUrl"`
		NewPoolJobIntervalSec   int64                    `json:"newPoolJobIntervalSec"`
		ReserveJobInterval      duration.Duration        `json:"reserveJobInterval"`
		UpdateReserveBulk       int                      `json:"updateReserveBulk"`
		ConcurrentBatches       int                      `json:"concurrentBatches"`
		PredefineSupportedPairs []*valueobject.TokenPair `json:"predefineSupportedPairs"`
	}
)

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return
	}

	err = json.Unmarshal(bodyBytes, &properties)
	if err != nil {
		return
	}

	if properties.UpdateReserveBulk == 0 {
		properties.UpdateReserveBulk = defaultUpdateReserveBulk
	}

	if properties.ReserveJobInterval.Duration == 0 {
		properties.ReserveJobInterval.Duration = defaultReserveJobInterval
	}
	if properties.ConcurrentBatches == 0 {
		properties.ConcurrentBatches = constant.DefaultConcurrentBatches
	}

	return
}
