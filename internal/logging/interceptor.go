package logging

import (
	"context"
	"log/slog"
	"time"

	mcpv1 "github.com/beduardo/eve-realm-mcp/gen/proto/mcp/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func NewInterceptor(logger *slog.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)

		st, _ := status.FromError(err)
		code := st.Code()

		attrs := []slog.Attr{
			slog.String("method", info.FullMethod),
			slog.Float64("duration_ms", float64(duration.Microseconds())/1000.0),
			slog.String("status", code.String()),
		}

		if info.FullMethod == mcpv1.MCPService_InvokeTool_FullMethodName {
			if invokeReq, ok := req.(*mcpv1.InvokeToolRequest); ok {
				attrs = append(attrs, slog.String("tool_name", invokeReq.ToolName))
			}
		}

		if code != codes.OK {
			attrs = append(attrs, slog.String("error", err.Error()))
			logger.LogAttrs(ctx, slog.LevelError, "rpc completed", attrs...)
		} else {
			logger.LogAttrs(ctx, slog.LevelInfo, "rpc completed", attrs...)
		}

		return resp, err
	}
}
