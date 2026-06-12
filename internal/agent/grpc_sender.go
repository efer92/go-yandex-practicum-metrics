package agent

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	pb "github.com/efer92/go-yandex-practicum-metrics/internal/proto"
)

// GRPCSender ships metric batches to the server over gRPC. It implements the
// same BatchSender contract as the HTTP Sender.
type GRPCSender struct {
	conn    *grpc.ClientConn
	client  pb.MetricsClient
	localIP string
}

// NewGRPCSender dials addr (host:port, no scheme) and returns a ready sender.
// The connection is lazy — errors surface on the first SendBatch.
func NewGRPCSender(addr string) (*GRPCSender, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("grpc dial %q: %w", addr, err)
	}
	return &GRPCSender{
		conn:    conn,
		client:  pb.NewMetricsClient(conn),
		localIP: detectLocalIPv4(),
	}, nil
}

// SendBatch converts the batch to protobuf and calls UpdateMetrics, attaching
// the agent's IP in the x-real-ip metadata for trusted-subnet checks.
func (s *GRPCSender) SendBatch(metrics []model.Metrics) error {
	req := &pb.UpdateMetricsRequest{Metrics: make([]*pb.Metric, 0, len(metrics))}
	for _, m := range metrics {
		pm := &pb.Metric{Id: m.ID}
		switch m.MType {
		case model.Gauge:
			if m.Value == nil {
				continue
			}
			pm.Type = pb.Metric_GAUGE
			pm.Value = *m.Value
		case model.Counter:
			if m.Delta == nil {
				continue
			}
			pm.Type = pb.Metric_COUNTER
			pm.Delta = *m.Delta
		default:
			continue
		}
		req.Metrics = append(req.Metrics, pm)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if s.localIP != "" {
		ctx = metadata.AppendToOutgoingContext(ctx, httpconst.MetadataXRealIP, s.localIP)
	}

	if _, err := s.client.UpdateMetrics(ctx, req); err != nil {
		return fmt.Errorf("grpc update metrics: %w", err)
	}
	return nil
}

// Close releases the underlying connection.
func (s *GRPCSender) Close() error {
	return s.conn.Close()
}
