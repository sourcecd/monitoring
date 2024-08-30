package server

import (
	"context"
	"log"
	"net"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"github.com/sourcecd/monitoring/internal/models"
	monproto "github.com/sourcecd/monitoring/proto"
)

type MonitoringServer struct {
	monproto.UnimplementedMonitoringServer
	mh *metricHandlers
}

// SendMetrics grpc method for send metrics
func (m *MonitoringServer) SendMetrics(ctx context.Context, in *monproto.MetricsRequest) (*monproto.MetricResponse, error) {
	var metrics []models.Metrics
	for _, metric := range in.Metric {
		metrics = append(metrics, models.Metrics{
			ID:    metric.Id,
			Value: &metric.Value,
			Delta: &metric.Delta,
			MType: metric.Mtype,
		})
	}
	if err := m.mh.reqRetrier.UseRetrierWMB(m.mh.storage.WriteBatchMetrics)(m.mh.ctx, metrics); err != nil {
		log.Println(err)
	}
	return &monproto.MetricResponse{
		Error: "OK",
	}, nil
}

// ListenGrpc method for accept grpc messages
func ListenGrpc(grpcServer string, mh *metricHandlers) error {
	cfg := zap.NewProductionConfig()
	zapLogger, _ := cfg.Build()

	grpc_zap.ReplaceGrpcLoggerV2(zapLogger)

	l, err := net.Listen("tcp", grpcServer)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_zap.UnaryServerInterceptor(zapLogger),
			grpc_recovery.UnaryServerInterceptor(),
		),
	)
	go func() {
		<-mh.ctx.Done()
		s.GracefulStop()
	}()
	monproto.RegisterMonitoringServer(s, &MonitoringServer{mh: mh})
	if err := s.Serve(l); err != nil {
		return err
	}
	return nil
}
