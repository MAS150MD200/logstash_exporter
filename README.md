# Logstash exporter

Prometheus metric exporter for logstash message flow. This exporter is written in Go.

## How it works

The exporter connects to a local or remote redis server and periodically polls a list.
As soon as a new event enter the list, the exporter picks this entry and increment the metric about processed events. Additionally a processing time will be calculated based on the original event timestamp and the current time. The difference between these two times is the latency for the logstash stack.

ELK stack overview and exporter:

   staging logstash -> redis (queue "logstash-staging") -> logstash (filters/grok/etc.) -> elasticsearch
                                                           -> redis (queue "logstash-prometheus-stats") => number of events/logstash duration
                       <= logstash_exporter: count elements in logstash-staging queue >10 problem with second logstash instance.


## Configure logstash

Staging Logstash:

   output {
      redis {
         host => ["127.0.0.1"]
	 key => "logstash-staging"
	 data_type => "list"
      }
   }
								    

Logstash
    input {
       redis {
          host => ["127.0.0.1"]
	  key => "logstash-staging"
	  data_type => "list"
       }
    }
    filter {
       ...
    }
    output {
       elasticsearch {
          ...
       }

       # write the stats at the end, to now the total latency
       redis {
          host => ["127.0.0.1"]
	  key => "logstash-prometheus-stats"
	  data_type => "list"
       }

    }


## Building and running

    go build logstash_exporter
    ./logstash_exporter




