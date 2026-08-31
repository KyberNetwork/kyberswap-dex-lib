package abi

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

// MustParseABI parses ABI JSON, panicking if it is malformed.
func MustParseABI[T ~string | ~[]byte](data T) abi.ABI {
	parsed, err := abi.JSON(abiReader(data))
	if err != nil {
		panic(err)
	}

	return parsed
}

// abiReader avoids copying the common []byte case; anything else goes through a
// string conversion, which also covers named types over string or []byte.
func abiReader[T ~string | ~[]byte](data T) io.Reader {
	if b, ok := any(data).([]byte); ok {
		return bytes.NewReader(b)
	}

	return strings.NewReader(string(data))
}

// MustABIMethod returns the named method, panicking if the ABI has no such name.
func MustABIMethod(parsed abi.ABI, name string) abi.Method {
	method, ok := parsed.Methods[name]
	if !ok {
		panic(missingABIEntry("method", name, methodNames(parsed)))
	}

	return method
}

// MustABIError returns the named custom error, panicking if the ABI has no such name.
func MustABIError(parsed abi.ABI, name string) abi.Error {
	abiErr, ok := parsed.Errors[name]
	if !ok {
		panic(missingABIEntry("error", name, errorNames(parsed)))
	}

	return abiErr
}

// MustABIEvent returns the named event, panicking if the ABI has no such name.
func MustABIEvent(parsed abi.ABI, name string) abi.Event {
	event, ok := parsed.Events[name]
	if !ok {
		panic(missingABIEntry("event", name, eventNames(parsed)))
	}

	return event
}

// missingABIEntry lists what the ABI does have, since the usual cause is a
// rename or reading one contract's name against another contract's ABI.
func missingABIEntry(kind, name string, have []string) string {
	sort.Strings(have)

	return fmt.Sprintf("abi: no %s %q; have: %s", kind, name, strings.Join(have, ", "))
}

func methodNames(parsed abi.ABI) []string {
	names := make([]string, 0, len(parsed.Methods))
	for name := range parsed.Methods {
		names = append(names, name)
	}

	return names
}

func errorNames(parsed abi.ABI) []string {
	names := make([]string, 0, len(parsed.Errors))
	for name := range parsed.Errors {
		names = append(names, name)
	}

	return names
}

func eventNames(parsed abi.ABI) []string {
	names := make([]string, 0, len(parsed.Events))
	for name := range parsed.Events {
		names = append(names, name)
	}

	return names
}
