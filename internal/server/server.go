package server

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	//"strings"

	"github.com/go-chi/chi/v5"
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
		// not needed for chi - 405 Method Not Allowed returns
		/*if req.Method != http.MethodPost {
			http.Error(resp, fmt.Sprintf("http method %s not acceptable for update metrics", req.Method), http.StatusBadRequest)
			return
		}*/
		if req.Header.Get("Content-Type") != "" && req.Header.Get("Content-Type") != "text/plain" {
			http.Error(resp, fmt.Sprintf("wrong content type: %s", req.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}

		// not needed for chi
		/*actionURL := req.URL.Path
		urlSplit := strings.Split(strings.Trim(actionURL, "/"), "/")
		if len(urlSplit) != 4 {
			http.Error(resp, "url does not match patter: /update/[metric_type]/[metric_name]/value", http.StatusNotFound)
			return
		}*/

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
				http.Error(resp, "can't store counter metric", http.StatusBadRequest)
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
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case "counter":
			val, err := storage.GetCounter(mVal)
			if err != nil {
				http.Error(resp, "counter not found", http.StatusNotFound)
				return
			}
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
		//_, _ = w.Write([]byte(storage.GetAllMetricsTxt()))
		_, _ = io.WriteString(w, fmt.Sprintf(htmlBasic, storage.GetAllMetricsTxt()))
	}
}

func chiRouter(storage storage.StoreMetrics) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", updateMetrics(storage))
	r.Get("/value/{type}/{val}", getMetrics(storage))
	r.Get("/", getAll(storage))

	return r
}

func Run(serverAddr string) {
	m := &storage.MemStorage{}
	m.Setup()

	log.Fatal(http.ListenAndServe(serverAddr, chiRouter(m)))
}
