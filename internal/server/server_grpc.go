package server

import (
	"context"
	"log"
	"net"
	"net/netip"

	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"google.golang.org/grpc"

	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip" // Install the gzip compressor
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	"github.com/sourcecd/monitoring/internal/models"
	monproto "github.com/sourcecd/monitoring/proto"
)

type MonitoringServer struct {
	monproto.UnimplementedMonitoringServer
	mh      *metricHandlers
	subnets []netip.Prefix
}

// SendMetrics grpc method for send metrics
func (m *MonitoringServer) SendMetrics(ctx context.Context, in *monproto.MetricsRequest) (*monproto.MetricResponse, error) {
	var (
		xrealip []string
		metrics []models.Metrics
	)
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		xrealip = md.Get("x-real-ip")
		grpc_ctxtags.Extract(ctx).Set("grpc-accept-encoding", md.Get("grpc-accept-encoding"))
		grpc_ctxtags.Extract(ctx).Set("x-real-ip", xrealip)
	}
	if m.subnets != nil {
		if len(xrealip) == 0 {
			return nil, status.Error(codes.PermissionDenied, "cant' find src ip")
		}
		ip, err := netip.ParseAddr(xrealip[0])
		if err != nil {
			return nil, status.Error(codes.PermissionDenied, "error parse src ip")
		}
		isIPtrue := false
		for _, v := range m.subnets {
			if v.Contains(ip) {
				isIPtrue = true
				break
			}
		}
		if !isIPtrue {
			return nil, status.Error(codes.PermissionDenied, "src ip not allowed")
		}
	}
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
func ListenGrpc(grpcServer string, subnets []netip.Prefix, mh *metricHandlers) error {
	cfg := zap.NewProductionConfig()
	zapLogger, _ := cfg.Build()

	grpc_zap.ReplaceGrpcLoggerV2(zapLogger)

	l, err := net.Listen("tcp", grpcServer)
	if err != nil {
		return err
	}

	s := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			grpc_ctxtags.UnaryServerInterceptor(),
			grpc_zap.UnaryServerInterceptor(zapLogger),
			grpc_recovery.UnaryServerInterceptor(),
		),
	)
	go func() {
		<-mh.ctx.Done()
		s.GracefulStop()
	}()
	monproto.RegisterMonitoringServer(s, &MonitoringServer{subnets: subnets, mh: mh})
	if err := s.Serve(l); err != nil {
		return err
	}
	return nil
}
