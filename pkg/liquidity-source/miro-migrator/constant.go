package miromigrator

import (
	"errors"
)

const (
	DexType = "miro-migrator"
)

var (
	defaultGas     int64 = 130000
	defaultReserve       = "100000000000000000000000000"

	ErrInvalidToken      = errors.New("invalid token")
	ErrMigrationIsPaused = errors.New("migration is paused")
)
