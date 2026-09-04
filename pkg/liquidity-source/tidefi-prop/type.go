package tidefiprop

// StaticExtra is stored in entity.Pool.StaticExtra, once per pool. Address
// is fixed at discovery time -- TideFi has a single swapper contract shared
// by every pool on a chain.
type StaticExtra struct {
	Address string `json:"a"`
}
