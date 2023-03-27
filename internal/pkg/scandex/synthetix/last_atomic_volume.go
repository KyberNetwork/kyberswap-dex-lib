package synthetix

import "math/big"

type ExchangeVolumeAtPeriod struct {
	Time   uint64   `json:"time"`
	Volume *big.Int `json:"volume"`
}
