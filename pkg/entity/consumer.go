package entity

import "time"

type Consumer struct {
	Name    string
	Pending int64
	Idle    time.Duration
}
