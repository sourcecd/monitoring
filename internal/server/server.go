package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/monitoring/internal/logging"
	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/models"
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
		case metrictypes.GaugeType:
			fl64, err := strconv.ParseFloat(metric.metricValue, 64)
			if err != nil {
				http.Error(resp, "can't parse gauge metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteGauge(metric.metricName, metrictypes.Gauge(fl64)); err != nil {
				http.Error(resp, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
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
		case metrictypes.GaugeType:
			val, err := storage.GetGauge(mVal)
			if err != nil {
				http.Error(resp, "gauge not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case metrictypes.CounterType:
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

func updateMetricsJSON(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		resultParsedJSON := models.Metrics{}
		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		switch resultParsedJSON.MType {
		case metrictypes.GaugeType:
			if resultParsedJSON.Value == nil {
				http.Error(w, "no value of gauge metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteGauge(resultParsedJSON.ID, metrictypes.Gauge(*resultParsedJSON.Value)); err != nil {
				http.Error(w, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := enc.Encode(&resultParsedJSON); err != nil {
				http.Error(w, "can't create gauge json answer", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
			if resultParsedJSON.Delta == nil {
				http.Error(w, "no value of counter metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteCounter(resultParsedJSON.ID, metrictypes.Counter(*resultParsedJSON.Delta)); err != nil {
				http.Error(w, "can't store counter metric", http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			if err := enc.Encode(&resultParsedJSON); err != nil {
				http.Error(w, "can't create counter json answer", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "bad metric type", http.StatusBadRequest)
			return
		}
	}
}

func getMetricsJSON(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		resultParsedJSON := models.Metrics{}
		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		switch resultParsedJSON.MType {
		case metrictypes.GaugeType:
			res, err := storage.GetGauge(resultParsedJSON.ID)
			if err != nil {
				http.Error(w, "gauge metric not found", http.StatusNotFound)
				return
			}
			resultParsedJSON.Value = (*float64)(&res)
			w.WriteHeader(http.StatusOK)
			if err := enc.Encode(&resultParsedJSON); err != nil {
				http.Error(w, "can't create gauge json", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
			res, err := storage.GetCounter(resultParsedJSON.ID)
			if err != nil {
				http.Error(w, "counter metric not found", http.StatusNotFound)
				return
			}
			resultParsedJSON.Delta = (*int64)(&res)
			w.WriteHeader(http.StatusOK)
			if err := enc.Encode(&resultParsedJSON); err != nil {
				http.Error(w, "can't create counter json", http.StatusInternalServerError)
				return
			}
		default:
			http.Error(w, "bad metric type", http.StatusBadRequest)
			return
		}
	}
}

func chiRouter(storage storage.StoreMetrics) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", logging.WriteLogging(updateMetrics(storage)))
	r.Get("/value/{type}/{val}", logging.WriteLogging(getMetrics(storage)))
	r.Get("/", logging.WriteLogging(getAll(storage)))

	//json
	r.Post("/update/", logging.WriteLogging(updateMetricsJSON(storage)))
	r.Post("/value/", logging.WriteLogging(getMetricsJSON(storage)))

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
