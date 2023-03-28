package makerpsm

import (
	"encoding/json"

	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type ScannerProperties struct {
	ConfigPath         string
	ReserveJobInterval duration.Duration
}

func NewScannerProperties(data map[string]interface{}) (ScannerProperties, error) {
	bodyBytes, err := json.Marshal(data)
	if err != nil {
		return ScannerProperties{}, err
	}

	var properties ScannerProperties
	if err = json.Unmarshal(bodyBytes, &properties); err != nil {
		return ScannerProperties{}, err
	}

	return properties, nil
}
