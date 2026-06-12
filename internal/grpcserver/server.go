// Package grpcserver exposes the metrics service over gRPC: a Metrics service
// implementation backed by service.MetricService and a trusted-subnet
// UnaryInterceptor mirroring the HTTP middleware behaviour.
package grpcserver

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	pb "github.com/efer92/go-yandex-practicum-metrics/internal/proto"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

// MetricsServer implements pb.MetricsServer on top of MetricService.
type MetricsServer struct {
	pb.UnimplementedMetricsServer
	svc *service.MetricService
}

// NewMetricsServer returns a MetricsServer backed by svc.
func NewMetricsServer(svc *service.MetricService) *MetricsServer {
	return &MetricsServer{svc: svc}
}

// UpdateMetrics applies the batch of metrics from the request.
func (s *MetricsServer) UpdateMetrics(ctx context.Context, req *pb.UpdateMetricsRequest) (*pb.UpdateMetricsResponse, error) {
	metrics := make([]model.Metrics, 0, len(req.GetMetrics()))
	for _, m := range req.GetMetrics() {
		switch m.GetType() {
		case pb.Metric_GAUGE:
			v := m.GetValue()
			metrics = append(metrics, model.Metrics{ID: m.GetId(), MType: model.Gauge, Value: &v})
		case pb.Metric_COUNTER:
			d := m.GetDelta()
			metrics = append(metrics, model.Metrics{ID: m.GetId(), MType: model.Counter, Delta: &d})
		default:
			return nil, status.Errorf(codes.InvalidArgument, "unknown metric type %v for %q", m.GetType(), m.GetId())
		}
	}
	if err := s.svc.UpdateBatch(ctx, metrics); err != nil {
		return nil, status.Errorf(codes.Internal, "update batch: %v", err)
	}
	return &pb.UpdateMetricsResponse{}, nil
}

// TrustedSubnetInterceptor returns a UnaryInterceptor that rejects calls whose
// x-real-ip metadata value is absent or outside the given CIDR with
// codes.PermissionDenied. Returns (nil, nil) when cidr is empty.
func TrustedSubnetInterceptor(cidr string) (grpc.UnaryServerInterceptor, error) {
	if cidr == "" {
		return nil, nil
	}
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return nil, fmt.Errorf("trusted_subnet: invalid CIDR %q: %w", cidr, err)
	}
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		var raw string
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			if vals := md.Get(httpconst.MetadataXRealIP); len(vals) > 0 {
				raw = vals[0]
			}
		}
		ip := net.ParseIP(raw)
		if ip == nil || !network.Contains(ip) {
			return nil, status.Errorf(codes.PermissionDenied, "ip %q is not in trusted subnet", raw)
		}
		return handler(ctx, req)
	}, nil
}
