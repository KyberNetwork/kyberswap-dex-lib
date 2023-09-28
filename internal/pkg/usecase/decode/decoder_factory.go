package decode

type IDecoder interface {
	Decode(data string) (interface{}, error)
}
