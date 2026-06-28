// Package mcp implements the MCPServiceServer gRPC interface, bridging
// incoming RPC calls to the internal ToolRegistry.
package mcp

import (
	"context"
	"errors"
	"fmt"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	"github.com/beduardo/eve-realm-mcp/internal/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// MCPService satisfies the generated MCPServiceServer interface. It delegates
// all tool management to a ToolRegistry supplied at construction time.
type MCPService struct {
	mcpv1.UnimplementedMCPServiceServer
	reg registry.ToolRegistry
}

// NewMCPService constructs an MCPService that uses reg as its backing store.
func NewMCPService(reg registry.ToolRegistry) *MCPService {
	return &MCPService{reg: reg}
}

// ListTools returns a ListToolsResponse containing one ToolDescriptor for each
// tool currently registered in the registry.
func (s *MCPService) ListTools(_ context.Context, _ *mcpv1.ListToolsRequest) (*mcpv1.ListToolsResponse, error) {
	tools := s.reg.List()
	descriptors := make([]*mcpv1.ToolDescriptor, 0, len(tools))
	for _, t := range tools {
		descriptors = append(descriptors, &mcpv1.ToolDescriptor{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.InputSchema,
		})
	}
	return &mcpv1.ListToolsResponse{Tools: descriptors}, nil
}

// InvokeTool dispatches the request to the named tool's handler. If the tool
// is not registered, it returns a gRPC NOT_FOUND status whose message contains
// the requested tool name.
func (s *MCPService) InvokeTool(_ context.Context, req *mcpv1.InvokeToolRequest) (*mcpv1.InvokeToolResponse, error) {
	output, err := s.reg.Invoke(req.ToolName, req.Input)
	if err != nil {
		var nfe *registry.NotFoundError
		if errors.As(err, &nfe) {
			return nil, status.Error(codes.NotFound, fmt.Sprintf("tool %q not found", nfe.ToolName))
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &mcpv1.InvokeToolResponse{Output: output}, nil
}
