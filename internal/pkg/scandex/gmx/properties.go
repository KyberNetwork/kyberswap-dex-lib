package gmx

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type Properties struct {
	AddressesPath      string
	ReserveJobInterval duration.Duration `json:"reserveJobInterval"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, _ := json.Marshal(data)
	err = json.Unmarshal(bodyBytes, &properties)

	return
}
