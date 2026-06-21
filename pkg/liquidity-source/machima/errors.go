package machima

import "errors"

var (
	ErrAntiSniperActive = errors.New("pool is in anti-sniper window")
	ErrInvalidPair      = errors.New("invalid pair: exactly one side must be a counter asset")
)
