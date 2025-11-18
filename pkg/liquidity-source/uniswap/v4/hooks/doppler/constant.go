package doppler

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	NoopHookAddresses = []common.Address{
		common.HexToAddress("0x45178A8D6d368D612B7552B217802b7F97262000"), // base migrator
		common.HexToAddress("0x53C050d3B09C80024138165520Bd7c078D9e2000"), // unichain migrator
		common.HexToAddress("0x892D3C2B4ABEAAF67d52A7B29783E2161B7CaD40"), // base multi-curve
	}

	ScheduledHookAddresses = []common.Address{
		common.HexToAddress("0x3e342a06f9592459D75721d6956B570F02eF2Dc0"), // base
		common.HexToAddress("0x580ca49389d83b019d07E17e99454f2F218e2dc0"), // monad
	}

	ErrCannotSwapBeforeStartingTime = errors.New("cannot swap before starting time")
)
