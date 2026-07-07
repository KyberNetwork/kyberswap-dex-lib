package clanker

import "errors"

var (
	ErrPoolSimIsNil     = errors.New("poolSim is nil")
	ErrPoolIsNotTracked = errors.New("pool is not tracked")
)
