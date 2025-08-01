package shared

type IWrapMetadata interface {
	GetWrapToken() string
	GetUnwrapToken() string
	GetPool() string
	GetHook() string
	GetTickSpacing() int32
	GetFee() uint32
	GetHookData() []byte
	IsUnwrapNative() bool
}
