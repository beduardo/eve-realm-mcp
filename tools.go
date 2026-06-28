//go:build tools

// Package tools pins build-time tool dependencies so that go mod tidy retains
// them as direct dependencies in go.mod. The packages listed here are imported
// by protoc-generated code (google.golang.org/grpc, google.golang.org/protobuf)
// and by the dual-server startup routine (golang.org/x/sync/errgroup).
// This file is excluded from normal builds by the "tools" build tag.
package tools

import (
	_ "golang.org/x/sync/errgroup"
	_ "google.golang.org/grpc"
	_ "google.golang.org/protobuf/runtime/protoimpl"
)
