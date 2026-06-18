package tokentax

import (
	"github.com/KyberNetwork/ethrpc"
)

type StaticTracker struct {
	result Result
}

func NewStaticTracker(result Result) Tracker {
	return StaticTracker{result: result}
}

func (StaticTracker) AddTaxCalls(*ethrpc.Request) bool { return false }
func (t StaticTracker) TaxResult() Result              { return t.result }
