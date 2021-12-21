// nginx-log_exporter - exports metrics for Prometheus from nginx-log
package main

import (
	"exporterHTTPServer"
	"flag"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const version = "0.1"

// command line options default values
var listenAddress = "*"
var listenPort = 9113 // https://github.com/prometheus/prometheus/wiki/Default-port-allocations
var metricsPath = "/metrics"
var logFile = "/var/log/nginx/access.log"
var debugParse = false

// init logging
var debugLog = log.New(os.Stdout, "nginx-log_exporter: DEBUG\t", log.Ldate|log.Ltime|log.Lmsgprefix)
var infoLog = log.New(os.Stdout, "nginx-log_exporter: INFO\t", log.Ldate|log.Ltime|log.Lmsgprefix)
var errorLog = log.New(os.Stderr, "nginx-log_exporter: ERROR\t", log.Ldate|log.Ltime|log.Lmsgprefix)

func init() {
	infoLog.Println("Starting nginx-log-exporter version " + version)

	// get command line options
	flag.StringVar(&listenAddress, "listen-address", listenAddress, "ip-address where exporter listens connectioins from Prometheus '<ip-addr>|localhost|*|any'")
	flag.IntVar(&listenPort, "listen-port", listenPort, "port where exporter listens connections from Prometheus '1-65535'")
	flag.StringVar(&metricsPath, "metrics-path", metricsPath, "path after http://<listen-address>:[<listen-port]/ from where exporter returns metrics")
	flag.StringVar(&logFile, "log-file", logFile, "path and file to access log of nginx-log")
	flag.BoolVar(&debugParse, "debug-parse", debugParse, "enable verbosity output for parsing")
	flag.Parse()

	infoLog.Printf("listens on http://%s:%v%s", listenAddress, listenPort, metricsPath)
	infoLog.Printf("parsing log file: %s", logFile)

	if debugParse {
		debugLog.Println("debug-parse enabled")
	} else {
		debugLog.Println("debug-parse disabled")
	}
}

func main() {

	errorChan := make(chan error)
	srv := &exporterHTTPServer.Server{
		ListenAddress: listenAddress,
		ListenPort:    listenPort,
		MetricsPath:   metricsPath,
		ErrorChan:     errorChan,
		Handler:       promhttp.Handler(),
	}

	srv.Start()
	go parse(logFile, errorChan)

	for err := range errorChan {
		errorLog.Println(err)
	}

	infoLog.Println("Stop nginx-log-log-exporter")
	os.Exit(0)
}

func parse(fileName string, errorChan chan error) error {
	// start tailf log file
	t, err := tail.TailFile(fileName, tail.Config{
		Follow:    true,
		MustExist: true,
		ReOpen:    true,
	})
	if err != nil {
		errorChan <- err
		close(errorChan)
		return err
	}

	// init metrics variables
	requestsDuration := prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: "nginx",
			Subsystem: "requests",
			Name:      "duration",
			Help:      "Requests duration.",
		})
	prometheus.MustRegister(requestsDuration)

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nginx",
			Subsystem: "requests",
			Name:      "total",
			Help:      "How many HTTP requests processed.",
		},
		[]string{"country", "domain", "method", "code"},
	)
	prometheus.MustRegister(requestsTotal)

	// parsing lines from log file
	for line := range t.Lines {
		l := strings.Replace(line.Text, "  ", " ", -1) // remove double spaces
		l = strings.Replace(l, "  ", " ", -1)          // remove double spaces
		l = strings.Replace(l, "\"", "", -1)           // remove quotes

		words := strings.Split(l, " ")
		if len(words) < 16 {
			if debugParse {
				debugLog.Printf("can't parse string from log: %v", line.Text)
			}
			continue
		}

		country := words[1]
		country = strings.Replace(country, "(", "", -1)
		country = strings.Replace(country, ")", "", -1)
		if m, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z]$`, country); !m {
			if debugParse {
				debugLog.Printf("can't parse COUNTRY from log string: %v", line.Text)
			}
			country = "-"
		}

		domain := words[6]
		if m, _ := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9-]{1,61}[a-zA-Z0-9](?:\.[a-zA-Z]{2,})+$`, domain); !m {
			if debugParse {
				debugLog.Printf("can't parse DOMAIN from log string: %v", line.Text)
			}
			continue
		}

		method := words[7]
		if method != "GET" && method != "POST" {
			if debugParse {
				debugLog.Printf("can't parse METHOD from log string: %v", line.Text)
			}
			continue
		}

		code := words[10]
		if m, _ := regexp.MatchString(`^[2345]\d\d$`, code); !m {
			if debugParse {
				debugLog.Printf("can't parse CODE from log string: %v", line.Text)
			}
			continue
		}

		duration := words[12]
		if m, _ := regexp.MatchString(`^\d*\.\d*$`, duration); !m {
			if debugParse {
				debugLog.Printf("can't parse DURATION from log string: %v", line.Text)
			}
			continue
		}
		fduration, _ := strconv.ParseFloat(duration, 64)

		requestsDuration.Observe(fduration)
		requestsTotal.WithLabelValues(country, domain, method, code).Inc()

		if debugParse {
			debugLog.Printf("Parsed string from log: country=%s; domain=%s; method=%s code=%s duration=%s", country, domain, method, code, duration)
		}
	}
	return nil
}
