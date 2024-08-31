package agent

import (
	"context"
	"log"
	"time"

	grpc_retry "github.com/grpc-ecosystem/go-grpc-middleware/retry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	monproto "github.com/sourcecd/monitoring/proto"

	"google.golang.org/grpc/encoding/gzip" // Install the gzip compressor
)

type metricSender interface{}

var opts = []grpc_retry.CallOption{
	grpc_retry.WithMax(3),
	grpc_retry.WithBackoff(grpc_retry.BackoffExponential(1 * time.Second)),
}

// encodeProto function for protobuf metric encode.
func encodeProto(metrics *jsonModelsMetrics) (*monproto.MetricsRequest, error) {
	var metricsProto monproto.MetricsRequest
	metrics.RLock()
	defer metrics.RUnlock()

	for _, v := range metrics.jsonMetricsSlice {
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
	return &metricsProto, nil
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

// protobuf send
func protoSend(ctx context.Context, grpcServerHost string, metricsReq *monproto.MetricsRequest) (*monproto.MetricResponse, error) {
	conn, err := grpcConnector(grpcServerHost)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := monproto.NewMonitoringClient(conn)
	resp, err := c.SendMetrics(ctx, metricsReq)
	if err != nil {
		return nil, err
	}
	return resp, nil
}
