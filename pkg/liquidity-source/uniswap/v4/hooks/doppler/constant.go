package doppler

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

var (
	h = common.HexToAddress

	NoopHookAddresses = []common.Address{
		h("0x4053D4fa966cbdCC20Ec62070aC8814De8bEE500"), // ethereum UniswapV4MigratorHook
		h("0x53C050d3B09C80024138165520Bd7c078D9e2000"), // unichain UniswapV4MigratorHook
		h("0x3E4c689BBf33b37106eBC13Db8aa5BF13a25e500"), // monad UniswapV4MigratorHook
		h("0x45178A8D6d368D612B7552B217802b7F97262000"), // base UniswapV4MigratorHook
		h("0x892D3C2B4ABEAAF67d52A7B29783E2161B7CaD40"), // base UniswapV4MulticurveInitializerHook
		h("0xbB7784A4d481184283Ed89619A3e3ed143e1Adc0"), // base DecayMulticurveInitializerHook
		h("0xD6FECFF347c6203A41874e8D77dE669B54e7A500"), // base UniswapV4MigratorHook
	}

	ScheduledHookAddresses = []common.Address{
		h("0xc6a562cb5CbFA29BCB1bDCCF903b8B8f2E4A2DC0"), // ethereum UniswapV4ScheduledMulticurveInitializerHook
		h("0x580ca49389d83b019d07E17e99454f2F218e2dc0"), // monad UniswapV4ScheduledMulticurveInitializerHook
		h("0x3e342a06f9592459D75721d6956B570F02eF2Dc0"), // base UniswapV4ScheduledMulticurveInitializerHook
	}

	InitializerAddresses = []common.Address{
		h("0xAA096F558f3d4c9226De77E7Cc05f18E180B2544"), // DopplerHookInitializer
		h("0xBDF938149ac6a781F94FAa0ed45E6A0e984c6544"), // DopplerHookInitializer
	}

	DHooks = map[common.Address]func(json.RawMessage) IDHook{ // Doppler Hooks i.e. DopplerHookInitializer's internal hooks
		h("0x97cAD5684FB7Cc2bEd9a9b5eBfba67138F4f2503"): NewRehypeDHook, // RehypeDopplerHook
		h("0x3Ec4798A9B11e8243A8Db99687f7A23597B96623"): NewRehypeDHook, // RehypeDopplerHook
		h("0xBF4195ab0B03e1eB3345dd1e83BeD7650b1ed123"): NewRehypeDHook, // RehypeDopplerHookInitializer TODO decay
	}

	ErrCannotSwapBeforeStartingTime = errors.New("cannot swap before starting time")
)
