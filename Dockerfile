FROM gliderlabs/alpine
MAINTAINER Thorsten Sauter <tsauter@gmx.net>

COPY logstash_exporter /bin/logstash_exporter

EXPOSE 8081
ENTRYPOINT [ "/bin/logstash_exporter" ]

