package server

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/sourcecd/monitoring/proto"
	"google.golang.org/grpc"
)

type MonitoringServer struct {
	monproto.UnimplementedMonitoringServer
}

func (m *MonitoringServer) SendMetrics(ctx context.Context, in *monproto.MetricsRequest) (*monproto.MetricResponse, error) {
	fmt.Println(in)
	return &monproto.MetricResponse{
		Error: "OK",
	}, nil
}

func ListenGrpc() {
	l, err := net.Listen("tcp", ":8989")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	monproto.RegisterMonitoringServer(s, &MonitoringServer{})
	if err := s.Serve(l); err != nil {
		log.Fatal(err)
	}
}