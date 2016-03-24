package main

import (
	"flag"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/garyburd/redigo/redis"
	"encoding/json"
	"time"
	"log"
	"log/syslog"
)

var (
	addr = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	redis_queue = flag.String("redis-queue", "logstash-stats", "Message queue between logstash and the exporter.")
	debug = flag.Bool("debug", false, "Enable debug logging.")

	logger *log.Logger
	
	processedLogEntries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "logstash",
			Subsystem: "exporter",
			Name: "events_processed_total",
			Help: "Total number of events processed by logstash.",
		},
		[]string{"host", "type"},
	)
	lastLogEntry = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "logstash",
			Subsystem: "exporter",
			Name: "last_seen_event",
			Help: "Timestamp of the last seen event in the redis queue.",
		},
		[]string{"host", "type"},
	)
	parsingDurationHistogram = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "logstash",
			Subsystem: "exporter",
			Name: "parsing_durations_histogram_seconds",
			Help: "Logstash parsing latency.",
			Buckets: prometheus.LinearBuckets(10-5*10, .5*10, 20),
		},
		[]string{"host", "type"},
	)
)

func init() {
	prometheus.MustRegister(processedLogEntries)
	prometheus.MustRegister(lastLogEntry)
	prometheus.MustRegister(parsingDurationHistogram)
}

func parsingAndUpdating(raw string) error {

	// Try to treat raw string as json text
	var m map[string]interface{}
	err := json.Unmarshal([]byte(raw), &m)
	if err != nil {
		logger.Printf("JSON parser failed for %s: %s\n", raw, err)
		return err
	}
	if(*debug) {
		logger.Printf("Raw string evaluated as %#v", m)
	}

	// Extract host and type from json
	host_raw := m["host"]
	if host_raw == nil {
		host_raw = "n/a"
	}
	host := host_raw.(string)
	typ_raw := m["type"]
	if typ_raw == nil {
		typ_raw = "n/a"
	}
	typ := typ_raw.(string)

	// Incement the Prometheus counter about total processed messages.
	processedLogEntries.WithLabelValues(host, typ).Add(1)
			
	// Extract the last timestamp from logstash entry
	timestamp := time.Now().UTC()
	json_timestamp := m["@timestamp"]
	if json_timestamp != nil {
		timestamp, err = time.Parse(time.RFC3339, json_timestamp.(string))
		if err != nil {
			logger.Printf("Unable to parse timestamp: %s\n", err)
			timestamp = time.Now().UTC()
		}
	}

	parsingDurationHistogram.WithLabelValues(host, typ).Observe(
		float64(time.Since(timestamp)) / float64(time.Second),
	)

	// Set the last seen timestamp
	lastLogEntry.WithLabelValues(host, typ).Set(float64(timestamp.Unix()))

	return nil
}

func main() {
	flag.Parse()

	// Initialize the system logger
	var err error
	logger, err = syslog.NewLogger(syslog.LOG_INFO, 0)
	if err != nil {
		log.Fatalf("Failed to initialize the system logger: %s\n", err)
	}

	// Execute a separate go function for a non blocking webservice
	go func() {
		defer logger.Fatalf("HTTP server died unexpected.")
		
		logger.Printf("Starting HTTP server on %s...\n", *addr)
		http.Handle("/metrics", prometheus.Handler())
		err := http.ListenAndServe(*addr, nil)
		if err != nil {
			logger.Fatalf("Failed to start HTTP server on %s: %s", *addr, err.Error())
		}
	}()

	for {
		logger.Printf("Start Redis queue listener...")
		
		client, err := redis.Dial("tcp", ":6379")
		if err != nil {
			panic(err)
		}
		defer client.Close()

		for {
			if(*debug) {
				logger.Printf("Waiting for new entry in Redis list queue \"%s\"...", *redis_queue)
			}

			val, err := redis.Strings(client.Do("BLPOP", *redis_queue, "0"))
			if(err != nil) {
				sleepTime := 5
				logger.Printf("Failed to retreive keys from Redis: %s. Sleeping for %d secs.\n",
					err.Error(), sleepTime)
				time.Sleep(time.Duration(sleepTime) * time.Second)
				continue
			}
			if len(val) != 2 {
				logger.Printf("Unable to convert Redis field value: %s\n", val)
				continue
			}

			if(*debug) {
				logger.Printf("Processing new queue entry: %#v", val)
			}
			parsingAndUpdating(val[1])
			if(*debug) {
				logger.Printf("Processing of queue entry finished")
			}
		}

		sleepTime := 10
		logger.Printf("Main queue ended. Sleeping for %d seconds.\n", sleepTime)
		time.Sleep(time.Duration(10) * time.Second)
	}

}
