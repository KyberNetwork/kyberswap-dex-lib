package tidefiprop

// StaticExtra is stored in entity.Pool.StaticExtra, once per pool. Address
// is fixed at discovery time -- TideFi has a single swapper contract shared
// by every pool on a chain, and is where quote()/swap() are called. Vault
// is the actual liquidity-holding/quoting engine Address forwards to: it's
// a hardcoded bytecode literal inside Address's contract (confirmed via
// decompilation), not readable through any getter, so it comes from config
// rather than being resolved on-chain.
type StaticExtra struct {
	Address string `json:"a"`
	Vault   string `json:"v"`
}
