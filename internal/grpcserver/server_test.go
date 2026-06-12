package grpcserver

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"github.com/efer92/go-yandex-practicum-metrics/internal/httpconst"
	"github.com/efer92/go-yandex-practicum-metrics/internal/model"
	pb "github.com/efer92/go-yandex-practicum-metrics/internal/proto"
	"github.com/efer92/go-yandex-practicum-metrics/internal/repository"
	"github.com/efer92/go-yandex-practicum-metrics/internal/service"
)

// startServer spins up an in-process gRPC server over bufconn and returns a
// connected client plus the backing storage for assertions.
func startServer(t *testing.T, trustedSubnet string) (pb.MetricsClient, *repository.MemStorage) {
	t.Helper()

	storage := repository.NewMemStorage()
	svc := service.NewMetricService(storage)

	var opts []grpc.ServerOption
	if trustedSubnet != "" {
		interceptor, err := TrustedSubnetInterceptor(trustedSubnet)
		require.NoError(t, err)
		opts = append(opts, grpc.UnaryInterceptor(interceptor))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterMetricsServer(srv, NewMetricsServer(svc))

	lis := bufconn.Listen(1 << 20)
	go srv.Serve(lis) //nolint:errcheck
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(_ context.Context, _ string) (net.Conn, error) {
			return lis.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })

	return pb.NewMetricsClient(conn), storage
}

func TestUpdateMetrics_GaugeAndCounter(t *testing.T) {
	client, storage := startServer(t, "")

	_, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{
			{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 3.14},
			{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 5},
		},
	})
	require.NoError(t, err)

	ctx := context.Background()
	g, ok := storage.GetMetric(ctx, "Alloc", model.Gauge)
	require.True(t, ok)
	assert.Equal(t, 3.14, *g.Value)

	c, ok := storage.GetMetric(ctx, "PollCount", model.Counter)
	require.True(t, ok)
	assert.Equal(t, int64(5), *c.Delta)
}

func TestUpdateMetrics_CounterAccumulates(t *testing.T) {
	client, storage := startServer(t, "")

	for i := 0; i < 3; i++ {
		_, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
			Metrics: []*pb.Metric{{Id: "PollCount", Type: pb.Metric_COUNTER, Delta: 2}},
		})
		require.NoError(t, err)
	}

	c, ok := storage.GetMetric(context.Background(), "PollCount", model.Counter)
	require.True(t, ok)
	assert.Equal(t, int64(6), *c.Delta)
}

func TestTrustedSubnetInterceptor_AllowsTrustedIP(t *testing.T) {
	client, _ := startServer(t, "10.0.0.0/8")

	ctx := metadata.AppendToOutgoingContext(context.Background(), httpconst.MetadataXRealIP, "10.1.2.3")
	_, err := client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}},
	})
	assert.NoError(t, err)
}

func TestTrustedSubnetInterceptor_RejectsUntrustedIP(t *testing.T) {
	client, _ := startServer(t, "10.0.0.0/8")

	ctx := metadata.AppendToOutgoingContext(context.Background(), httpconst.MetadataXRealIP, "192.168.1.1")
	_, err := client.UpdateMetrics(ctx, &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}},
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTrustedSubnetInterceptor_RejectsMissingMetadata(t *testing.T) {
	client, _ := startServer(t, "10.0.0.0/8")

	_, err := client.UpdateMetrics(context.Background(), &pb.UpdateMetricsRequest{
		Metrics: []*pb.Metric{{Id: "Alloc", Type: pb.Metric_GAUGE, Value: 1}},
	})
	require.Error(t, err)
	assert.Equal(t, codes.PermissionDenied, status.Code(err))
}

func TestTrustedSubnetInterceptor_EmptyCIDRDisabled(t *testing.T) {
	interceptor, err := TrustedSubnetInterceptor("")
	require.NoError(t, err)
	assert.Nil(t, interceptor)
}

func TestTrustedSubnetInterceptor_InvalidCIDR(t *testing.T) {
	_, err := TrustedSubnetInterceptor("bogus")
	assert.Error(t, err)
}
