package timer

import (
	"time"

	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

func Start(task interface{}) func() {
	logger.Infof("Start %v ...", task)

	start := time.Now()

	return func() {
		logger.Infof("Finish %v in: %v", task, time.Since(start))
	}
}
