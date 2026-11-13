package everlongflamm

import (
	"bytes"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

// The pool's, the Factory's, the Router's and a pricing hook's entries this source calls: discovery, the trade mask,
// the pricers, the price feed's address, the cheap lane's state, the quote and the request a swap hands its hook,
// the kinds of the pool's lending venues for the gas, and a hook's fixed parameters, `params()`, read as the 35
// words it answers. On a buy whose ladder carries a leverage label the hook's `levState` is read too, sent as raw
// call data on the request's words (levStateSelector), so it has no entry here; the sampler reads it the same way,
// and the curve's fee by the hook's `quotePair`. Of each live lending venue's account, `capabilities()` and
// `minimumDebt(id)` (accountABI), for the guard of a venue that holds a minimum position; nothing of any feed or
// lender is called. samplerABI is FlammSampler's, `TAG()` and `sample`: every refresh's probes go out as one `sample`
// call.
var poolABI, factoryABI, routerABI, hookABI, accountABI, samplerABI abi.ABI

// samplerRuntime is FlammSampler's runtime code, the code a state override sets at the taker.
var samplerRuntime = common.FromHex(strings.TrimSpace(samplerRuntimeHex))

func init() {
	for _, b := range []struct {
		dst  *abi.ABI
		data []byte
	}{{&poolABI, poolABIJson}, {&factoryABI, factoryABIJson}, {&routerABI, routerABIJson}, {&hookABI, hookABIJson},
		{&accountABI, accountABIJson}, {&samplerABI, samplerABIJson}} {
		parsed, err := abi.JSON(bytes.NewReader(b.data))
		if err != nil {
			panic(err)
		}
		*b.dst = parsed
	}
}
