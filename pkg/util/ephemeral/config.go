package ephemeral

// LensConfig is how a tracker reads its pools through its lens. A dex's Config embeds it, so its
// keys sit beside the dex's own, and says there what its lens reads.
type LensConfig struct {
	// UseLens reads the pools through the dex's lens.
	UseLens bool `json:"useLens,omitempty"`
	// LensGas is the gas each lens call may use; 0 takes the node's eth_call cap, up to
	// CallGasCeiling.
	LensGas uint64 `json:"lensGas,omitempty"`
	// LensGuard bounds the lens reads made while updating pools, which never wait for it.
	LensGuard GuardConfig `json:"lensGuard"`
	// LensBatch gathers the first lens read of pools read at the same block into one call, with
	// every exchange that reads through the same node; nil reads each pool alone.
	LensBatch *BatchConfig `json:"lensBatch,omitempty"`
}
