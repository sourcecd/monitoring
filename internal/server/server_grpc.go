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

func ListenGrpc(grpcServer string, mh *metricHandlers) {
	l, err := net.Listen("tcp", grpcServer)
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	monproto.RegisterMonitoringServer(s, &MonitoringServer{mh: mh})
	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
}
