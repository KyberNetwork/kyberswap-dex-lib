package clanker

import "errors"

var (
	ErrClankerCallerIsNil = errors.New("clanker caller is nil")
	ErrPoolSimIsNil       = errors.New("poolSim is nil")
)
