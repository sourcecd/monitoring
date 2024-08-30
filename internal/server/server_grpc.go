package server

import (
	"context"
	"log"
	"net"

	"github.com/sourcecd/monitoring/internal/models"
	"github.com/sourcecd/monitoring/proto"
	"google.golang.org/grpc"
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
	l, err := net.Listen("tcp", grpcServer)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
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
