package common

import (
	"reflect"

	"github.com/huandu/go-clone"
)

// NoDeepClone is a wrapper where "github.com/huandu/go-clone".Slowly() doesn't deepclone its wrapped object.
type NoDeepClone struct {
	Inner interface{}
}

// MakeNoClone wraps an object
func MakeNoClone(data interface{}) NoDeepClone {
	return NoDeepClone{
		Inner: data,
	}
}

// Get the wrapped object
func (c NoDeepClone) Get() interface{} {
	return c.Inner
}

func init() {
	clone.SetCustomFunc(reflect.TypeOf(NoDeepClone{}), func(allocator *clone.Allocator, old, new reflect.Value) {
		new.FieldByName("Inner").Set(old.FieldByName("Inner"))
	})
}
