package fxdx

// Code generated by github.com/tinylib/msgp DO NOT EDIT.

import (
	"github.com/tinylib/msgp/msgp"
)

// DecodeMsg implements msgp.Decodable
func (z *ChainlinkFlags) DecodeMsg(dc *msgp.Reader) (err error) {
	var zb0001 uint32
	zb0001, err = dc.ReadArrayHeader()
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	var zb0002 uint32
	zb0002, err = dc.ReadMapHeader()
	if err != nil {
		err = msgp.WrapError(err, "Flags")
		return
	}
	if z.Flags == nil {
		z.Flags = make(map[string]bool, zb0002)
	} else if len(z.Flags) > 0 {
		for key := range z.Flags {
			delete(z.Flags, key)
		}
	}
	for zb0002 > 0 {
		zb0002--
		var za0001 string
		var za0002 bool
		za0001, err = dc.ReadString()
		if err != nil {
			err = msgp.WrapError(err, "Flags")
			return
		}
		za0002, err = dc.ReadBool()
		if err != nil {
			err = msgp.WrapError(err, "Flags", za0001)
			return
		}
		z.Flags[za0001] = za0002
	}
	return
}

// EncodeMsg implements msgp.Encodable
func (z *ChainlinkFlags) EncodeMsg(en *msgp.Writer) (err error) {
	// array header, size 1
	err = en.Append(0x91)
	if err != nil {
		return
	}
	err = en.WriteMapHeader(uint32(len(z.Flags)))
	if err != nil {
		err = msgp.WrapError(err, "Flags")
		return
	}
	for za0001, za0002 := range z.Flags {
		err = en.WriteString(za0001)
		if err != nil {
			err = msgp.WrapError(err, "Flags")
			return
		}
		err = en.WriteBool(za0002)
		if err != nil {
			err = msgp.WrapError(err, "Flags", za0001)
			return
		}
	}
	return
}

// MarshalMsg implements msgp.Marshaler
func (z *ChainlinkFlags) MarshalMsg(b []byte) (o []byte, err error) {
	o = msgp.Require(b, z.Msgsize())
	// array header, size 1
	o = append(o, 0x91)
	o = msgp.AppendMapHeader(o, uint32(len(z.Flags)))
	for za0001, za0002 := range z.Flags {
		o = msgp.AppendString(o, za0001)
		o = msgp.AppendBool(o, za0002)
	}
	return
}

// UnmarshalMsg implements msgp.Unmarshaler
func (z *ChainlinkFlags) UnmarshalMsg(bts []byte) (o []byte, err error) {
	var zb0001 uint32
	zb0001, bts, err = msgp.ReadArrayHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err)
		return
	}
	if zb0001 != 1 {
		err = msgp.ArrayError{Wanted: 1, Got: zb0001}
		return
	}
	var zb0002 uint32
	zb0002, bts, err = msgp.ReadMapHeaderBytes(bts)
	if err != nil {
		err = msgp.WrapError(err, "Flags")
		return
	}
	if z.Flags == nil {
		z.Flags = make(map[string]bool, zb0002)
	} else if len(z.Flags) > 0 {
		for key := range z.Flags {
			delete(z.Flags, key)
		}
	}
	for zb0002 > 0 {
		var za0001 string
		var za0002 bool
		zb0002--
		za0001, bts, err = msgp.ReadStringBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "Flags")
			return
		}
		za0002, bts, err = msgp.ReadBoolBytes(bts)
		if err != nil {
			err = msgp.WrapError(err, "Flags", za0001)
			return
		}
		z.Flags[za0001] = za0002
	}
	o = bts
	return
}

// Msgsize returns an upper bound estimate of the number of bytes occupied by the serialized message
func (z *ChainlinkFlags) Msgsize() (s int) {
	s = 1 + msgp.MapHeaderSize
	if z.Flags != nil {
		for za0001, za0002 := range z.Flags {
			_ = za0002
			s += msgp.StringPrefixSize + len(za0001) + msgp.BoolSize
		}
	}
	return
}
