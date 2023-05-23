package entity

type GroupInfo struct {
	Name            string
	Consumers       int64
	Pending         int64
	LastDeliveredID string
	EntriesRead     int64
	Lag             int64
}
