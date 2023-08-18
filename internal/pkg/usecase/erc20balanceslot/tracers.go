package erc20balanceslot

import (
	"bytes"
	_ "embed"

	"github.com/tdewolff/minify/v2/js"
)

//go:embed sloadTracer.js
var sloadTracer []byte

var sloadTracerMinified []byte

type tracingResultSload struct {
	Address string `json:"addr"`
	Slot    string `json:"slot"`
	Value   string `json:"value"`
}

type tracingResult struct {
	Sloads []tracingResultSload `json:"sloads"`
	Output string               `json:"output"`
}

func init() {
	// we need to minify the tracer script because we can not put multipleline string in JSON value
	minified := new(bytes.Buffer)
	err := js.Minify(nil, minified, bytes.NewReader(sloadTracer), nil)
	if err != nil {
		panic(err)
	}
	sloadTracerMinified = bytes.TrimPrefix(minified.Bytes(), []byte("var tracer="))
}
