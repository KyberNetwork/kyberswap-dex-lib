package job

import (
	"github.com/KyberNetwork/kutils"
	"github.com/KyberNetwork/pool-service/pkg/message"
)

type BatchedPoolAddress = kutils.ChanTask[*message.EventMessage]
