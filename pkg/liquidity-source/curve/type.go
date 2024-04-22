package curve

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying"`

	TokenInIsNative  *bool
	TokenOutIsNative *bool
}
