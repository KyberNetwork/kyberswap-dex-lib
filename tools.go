//go:build tools
// +build tools

package tools

import (
	_ "go.uber.org/mock/gomock"
	_ "go.uber.org/mock/mockgen/model"
	_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	_ "google.golang.org/protobuf/cmd/protoc-gen-go"
)
