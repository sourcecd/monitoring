package server

import (
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/monitoring/internal/logging"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/storage"
	"go.uber.org/zap"
)

type urlToMetric struct {
	metricType  string
	metricName  string
	metricValue string
}

func updateMetrics(storage storage.StoreMetrics) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Content-Type") != "" && req.Header.Get("Content-Type") != "text/plain" {
			http.Error(resp, fmt.Sprintf("wrong content type: %s", req.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}

		metric := urlToMetric{
			metricType:  chi.URLParam(req, "type"),
			metricName:  chi.URLParam(req, "name"),
			metricValue: chi.URLParam(req, "value"),
		}

		switch metric.metricType {
		case "gauge":
			fl64, err := strconv.ParseFloat(metric.metricValue, 64)
			if err != nil {
				http.Error(resp, "can't parse gauge metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteGauge(metric.metricName, metrictypes.Gauge(fl64)); err != nil {
				http.Error(resp, "can't store gauge metric", http.StatusBadRequest)
				return
			}
		case "counter":
			i64, err := strconv.ParseInt(metric.metricValue, 10, 64)
			if err != nil {
				http.Error(resp, "can't parse counter metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteCounter(metric.metricName, metrictypes.Counter(i64)); err != nil {
				http.Error(resp, "can't store counter metric", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(resp, "metric_type not found", http.StatusBadRequest)
			return
		}

		resp.Header().Set("Content-Type", "text/plain")
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("OK"))
	}
}

func getMetrics(storage storage.StoreMetrics) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("Content-Type", "text/plain")
		mType := chi.URLParam(req, "type")
		mVal := chi.URLParam(req, "val")
		switch mType {
		case "gauge":
			val, err := storage.GetGauge(mVal)
			if err != nil {
				http.Error(resp, "gauge not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case "counter":
			val, err := storage.GetCounter(mVal)
			if err != nil {
				http.Error(resp, "counter not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		default:
			http.Error(resp, "metric_type not found", http.StatusBadRequest)
			return
		}
	}
}

func getAll(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		htmlBasic := `<!DOCTYPE html>
<html lang="en">
  <head>
    <meta charset="UTF-8" />
    <meta name="counters" content="width=device-width, initial-scale=1.0" />
    <title>counters</title>
  </head>
  <body>
  <pre>
%s
  </pre>
  </body>
</html>`
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, fmt.Sprintf(htmlBasic, storage.GetAllMetricsTxt()))
	}
}

func chiRouter(storage storage.StoreMetrics) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", logging.WriteLogging(updateMetrics(storage)))
	r.Get("/value/{type}/{val}", logging.WriteLogging(getMetrics(storage)))
	r.Get("/", logging.WriteLogging(getAll(storage)))

	return r
}

func Run(serverAddr, loglevel string) {

	if err := logging.Setup(loglevel); err != nil {
		panic(err)
	}

	m := &storage.MemStorage{}
	m.Setup()

	logging.Log.Info("Starting server on", zap.String("address", serverAddr))
	logging.Log.Fatal("Failed to start server", zap.Error(http.ListenAndServe(serverAddr, chiRouter(m))))
}
