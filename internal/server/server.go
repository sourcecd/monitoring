package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/sourcecd/monitoring/internal/compression"
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
			if err := storage.WriteMetric(metric.metricType, metric.metricName, metrictypes.Gauge(fl64)); err != nil {
				http.Error(resp, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		case metrictypes.CounterType:
			i64, err := strconv.ParseInt(metric.metricValue, 10, 64)
			if err != nil {
				http.Error(resp, "can't parse counter metric", http.StatusBadRequest)
				return
			}
			if err := storage.WriteMetric(metric.metricType, metric.metricName, metrictypes.Counter(i64)); err != nil {
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
			val, err := storage.GetMetric(metrictypes.GaugeType, mVal)
			if err != nil {
				http.Error(resp, "gauge not found", http.StatusNotFound)
				return
			}
			resp.WriteHeader(http.StatusOK)
			_, _ = io.WriteString(resp, fmt.Sprintf("%v\n", val))
		case metrictypes.CounterType:
			val, err := storage.GetMetric(metrictypes.CounterType, mVal)
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
		tmpl, _ := template.New("data").Parse(`
<!DOCTYPE html>
<html lang="en">
<head>
	<meta charset="UTF-8" />
	<meta name="counters" content="width=device-width, initial-scale=1.0" />
	<title>Counters</title>
</head>
<body>
<pre>
{{ .}}
</pre>
</body>
</html>`)
		res, err := storage.GetAllMetricsTxt()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_ = tmpl.Execute(w, res)
	}
}

func updateMetricsJSON(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resultParsedJSON models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		if resultParsedJSON.MType == metrictypes.GaugeType && resultParsedJSON.Value != nil {
			if err := storage.WriteMetric(resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Gauge(*resultParsedJSON.Value)); err != nil {
				log.Println(err)
				http.Error(w, "can't store gauge metric", http.StatusInternalServerError)
				return
			}
		} else if resultParsedJSON.MType == metrictypes.CounterType && resultParsedJSON.Delta != nil {
			if err := storage.WriteMetric(resultParsedJSON.MType, resultParsedJSON.ID, metrictypes.Counter(*resultParsedJSON.Delta)); err != nil {
				log.Println(err)
				http.Error(w, "can't store counter metric", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "bad metric type or no metric value", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(&resultParsedJSON); err != nil {
			http.Error(w, "can't prepare json answer", http.StatusInternalServerError)
			return
		}
	}
}

func getMetricsJSON(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var resultParsedJSON models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&resultParsedJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		// use test method (from mentor issue iter9)
		res, err := storage.GetMetric(resultParsedJSON.MType, resultParsedJSON.ID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		if g, ok := res.(metrictypes.Gauge); ok {
			resultParsedJSON.Value = (*float64)(&g)
		} else if c, ok := res.(metrictypes.Counter); ok {
			resultParsedJSON.Delta = (*int64)(&c)
		} else {
			http.Error(w, "bad metric type", http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(&resultParsedJSON); err != nil {
			http.Error(w, "can't encode json", http.StatusInternalServerError)
			return
		}
	}
}

func updateBatchMetricsJSON(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var batchMettricsJSON []models.Metrics

		if r.Header.Get("Content-Type") != "application/json" {
			http.Error(w, fmt.Sprintf("wrong content type: %s", r.Header.Get("Content-Type")), http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")

		dec := json.NewDecoder(r.Body)

		if err := dec.Decode(&batchMettricsJSON); err != nil {
			http.Error(w, "error to pasrse json request", http.StatusBadRequest)
			return
		}
		enc := json.NewEncoder(w)

		if err := storage.WriteBatchMetrics(batchMettricsJSON); err != nil {
			log.Println(err)
			http.Error(w, "error to store batch metrics", http.StatusInternalServerError)
			return
		}
		// check ref
		w.WriteHeader(http.StatusOK)
		if err := enc.Encode(batchMettricsJSON); err != nil {
			http.Error(w, "can't encode json", http.StatusInternalServerError)
			return
		}
	}
}

func dbPing(storage storage.StoreMetrics) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := storage.Ping(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK\n"))
	}
}

func chiRouter(storage storage.StoreMetrics) chi.Router {
	r := chi.NewRouter()

	r.Post("/update/{type}/{name}/{value}", logging.WriteLogging(compression.GzipCompDecomp(updateMetrics(storage))))
	r.Get("/value/{type}/{val}", logging.WriteLogging(compression.GzipCompDecomp(getMetrics(storage))))
	r.Get("/", logging.WriteLogging(compression.GzipCompDecomp(getAll(storage))))

	//json
	r.Post("/update/", logging.WriteLogging(compression.GzipCompDecomp(updateMetricsJSON(storage))))
	r.Post("/value/", logging.WriteLogging(compression.GzipCompDecomp(getMetricsJSON(storage))))
	r.Post("/updates/", logging.WriteLogging(compression.GzipCompDecomp(updateBatchMetricsJSON(storage))))

	//ping
	r.Get("/ping", logging.WriteLogging(compression.GzipCompDecomp(dbPing(storage))))

	return r
}

func saveToFile(m *storage.MemStorage, fname string, duration int) {
	for {
		time.Sleep(time.Second * time.Duration(duration))
		if err := m.SaveToFile(fname); err != nil {
			log.Println(err)
		}
		if duration == 0 {
			break
		}
	}
}

func Run(serverAddr, loglevel string, storeInterval int, fileStoragePath string, restore bool, sigs chan os.Signal, databaseDsn string) {
	if err := logging.Setup(loglevel); err != nil {
		log.Fatal(err)
	}

	var store storage.StoreMetrics

	if databaseDsn != "" {
		signal.Reset(syscall.SIGINT, syscall.SIGTERM)
		//PGDB new
		pgdb, err := storage.NewPgDB(databaseDsn)
		if err != nil {
			log.Fatal(err)
		}
		defer pgdb.CloseDB()

		//main context timeout (default 60 sec)
		pgdb.SetTimeout(30 * time.Second)
		pgdb.SetBackoff(1*time.Second, 3)

		if err := pgdb.PopulateDB(); err != nil {
			log.Fatal(err)
		}

		store = pgdb
	} else {
		m := storage.NewMemStorage()

		if restore {
			if err := m.ReadFromFile(fileStoragePath); err != nil {
				log.Println(err)
			}
		}

		//save result on shutdown and throw signal
		go func() {
			sig := <-sigs
			fmt.Println("saving")
			saveToFile(m, fileStoragePath, 0)
			fmt.Println(sig)
			signal.Reset(syscall.SIGINT, syscall.SIGTERM)
			pid := os.Getpid()
			p, err := os.FindProcess(pid)
			if err != nil {
				log.Fatal(err)
			}
			if err := p.Signal(sig); err != nil {
				log.Fatal(err)
			}
		}()

		go saveToFile(m, fileStoragePath, storeInterval)

		store = m
	}

	logging.Log.Info("Starting server on", zap.String("address", serverAddr))
	logging.Log.Fatal("Failed to start server", zap.Error(http.ListenAndServe(serverAddr, chiRouter(store))))
}
