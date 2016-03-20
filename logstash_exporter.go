package main

import (
	"fmt"
	"flag"
	"net/http"
	"github.com/prometheus/client_golang/prometheus"
	"gopkg.in/redis.v3"
	"encoding/json"
	"time"
)

var addr = flag.String("listen-address", ":8080", "The adress to listen on for HTTP requests.")

var (
	processedLogEntries = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "logstash_events_processed_total",
		Help: "xxx"})
	lastLogEntry = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "logstash_events_last_event",
		Help: "xxx"})
/*
	rpcDurationsHistogram = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "rpc_durations_histogram_microseconds",
		Help:    "RPC latency distributions.",
		Buckets: prometheus.LinearBuckets(10-5*10, .5*10, 20),
	})
*/
)

func init() {
	prometheus.MustRegister(processedLogEntries)
	prometheus.MustRegister(lastLogEntry)
	//prometheus.MustRegister(rpcDurationsHistogram)
}

func main() {
	go func() {
		fmt.Printf("--> Starting metric server...")
		http.Handle("/metrics", prometheus.Handler())
		err := http.ListenAndServe(*addr, nil)
		if err != nil {
			fmt.Printf("Failed to create metric server: %s\n", err.Error())
		}
	}()

	client := redis.NewClient(&redis.Options{
		Addr: "192.168.133.10:6379"})
	pong, err := client.Ping().Result()
	fmt.Println(pong, err)


	for {
		fmt.Printf("--> Start listening for new entires in the queue...\n")
		val, err := client.BLPop(0, "logstash_stats").Result()
		if(err != nil) {
			fmt.Printf("failed to retreive keys from redis: %s\n", err.Error())
		}

		fmt.Printf("--> Processing new entry in the queue.\n")
		fmt.Printf("Entry in queue: %s\n", val[1])

		var m map[string]interface{}
		err = json.Unmarshal([]byte(val[1]), &m)
		if err != nil {
			panic(fmt.Sprintf("Error: %s\n", err.Error()))
		}
		//fmt.Println(m)

		// Extract the last timestamp from logstash entry
		timestamp, err := time.Parse(time.RFC3339, m["@timestamp"].(string))
		if err != nil {
			fmt.Printf("Unable to parse timestamp: %s\n", err.Error())
			timestamp = time.Now().UTC()
		}
		lastLogEntry.Set(float64(timestamp.Unix()))
		

		// Incement the Prometheus counter about total processed messages.
		processedLogEntries.Inc()
	}

}
