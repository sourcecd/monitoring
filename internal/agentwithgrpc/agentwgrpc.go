// Package agentwithgrpc with agent grpc transport
package agentwithgrpc

import (
	"context"
	"log"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"

	"github.com/sourcecd/monitoring/internal/metrictypes"
	monproto "github.com/sourcecd/monitoring/proto"

	"google.golang.org/grpc/encoding/gzip" // Install the gzip compressor
)

var opts = []grpc_retry.CallOption{
	grpc_retry.WithMax(3),
	grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1 * time.Second)),
}

type MonMetricReq struct {
	MonProtoReq *monproto.MetricsRequest
}

func (m *MonMetricReq) Send(ctx context.Context, serverHost, xRealIp string) error {
	_, err := protoSend(ctx, serverHost, xRealIp, m.MonProtoReq)
	return err
}

// EncodeProto function for protobuf metric encode.
func EncodeProto(metrics *metrictypes.JSONModelsMetrics) (*MonMetricReq, error) {
	var metricsProto monproto.MetricsRequest
	metrics.RLock()
	defer metrics.RUnlock()

	for _, v := range metrics.JSONMetricsSlice {
		switch v.MType {
		case "counter":
			if v.Delta == nil {
				log.Println("nil value of metric counter")
				continue
			}
			metricsProto.Metric = append(metricsProto.Metric, &monproto.MetricsRequest_MetricRequest{
				Mtype: v.MType,
				Id:    v.ID,
				Delta: *v.Delta,
			})
		case "gauge":
			if v.Value == nil {
				log.Println("nil value of metric gauge")
				continue
			}
			metricsProto.Metric = append(metricsProto.Metric, &monproto.MetricsRequest_MetricRequest{
				Mtype: v.MType,
				Id:    v.ID,
				Value: *v.Value,
			})
		default:
			log.Println("unknown metric type")
		}
	}
	return &MonMetricReq{
		MonProtoReq: &metricsProto,
	}, nil
}

// grpc connect method
func grpcConnector(grpcServerHost string) (*grpc.ClientConn, error) {
	conn, err := grpc.NewClient(grpcServerHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithUnaryInterceptor(grpc_retry.UnaryClientInterceptor(opts...)),
	)
	if err != nil {
		return nil, err
	}
	return conn, nil
}

// ProtoSend send
func protoSend(ctx context.Context, grpcServerHost, xRealIp string, metricsReq *monproto.MetricsRequest) (*monproto.MetricResponse, error) {
	conn, err := grpcConnector(grpcServerHost)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := monproto.NewMonitoringClient(conn)
	md := metadata.New(map[string]string{"X-Real-IP": xRealIp})
	resp, err := c.SendMetrics(metadata.NewOutgoingContext(ctx, md), metricsReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
