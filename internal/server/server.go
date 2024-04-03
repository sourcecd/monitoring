package server

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/sourcecd/monitoring/internal/metrictypes"
	"github.com/sourcecd/monitoring/internal/storage"
)

type urlToMetric struct {
	metricType  string
	metricName  string
	metricValue string
}

func updateMetrics(storage storage.StoreMetrics) http.HandlerFunc {
	return func(resp http.ResponseWriter, req *http.Request) {
		if req.Method != http.MethodPost {
			http.Error(resp, fmt.Sprintf("http method %s not acceptable for update metrics", req.Method), http.StatusBadRequest)
			return
		}
		if req.Header.Get("Content-Type") != "" && req.Header.Get("Content-Type") != "text/plain" {
			http.Error(resp, fmt.Sprintf("wrong content type: %s", req.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}

		actionURL := req.URL.Path
		urlSplit := strings.Split(strings.Trim(actionURL, "/"), "/")
		if len(urlSplit) != 4 {
			http.Error(resp, "url does not match patter: /update/[metric_type]/[metric_name]/value", http.StatusNotFound)
			return
		}

		metric := urlToMetric{
			metricType:  urlSplit[1],
			metricName:  urlSplit[2],
			metricValue: urlSplit[3],
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
				http.Error(resp, "can't store counter metric", http.StatusBadRequest)
				return
			}
		default:
			http.Error(resp, "metric_type not found", http.StatusBadRequest)
			return
		}

		resp.Header().Set("Content-Type:", "text/plain")
		resp.WriteHeader(http.StatusOK)
		_, _ = resp.Write([]byte("OK"))
	}
}

// tmp func for test
func getAll(m *storage.MemStorage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprint(*m)))
	}
}

func Run() {
	m := &storage.MemStorage{}
	m.Setup()

	http.HandleFunc("/update/", updateMetrics(m))
	http.HandleFunc("/get", getAll(m))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
