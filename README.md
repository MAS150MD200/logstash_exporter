# Logstash exporter

Prometheus metric exporter for logstash message flow. This exporter is written in Go.

## How it works

The exporter connects to a local or remote redis server and periodically polls a list.
As soon as a new event enter the list, the exporter picks this entry and increment the metric about processed events. Additionally a processing time will be calculated based on the original event timestamp and the current time. The difference between these two times is the latency for the logstash stack.

## Configure logstash

    output {
    }


## Building and running

    go build logstash_exporter
    ./logstash_exporter




