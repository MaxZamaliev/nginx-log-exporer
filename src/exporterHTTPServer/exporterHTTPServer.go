// exporterHTTPServer - http server for metrics for Prometheus
package exporterHTTPServer

import (
    "context"
    "fmt"
    "net"
    "net/http"
    "os"
    "os/signal"
    "regexp"
    "strconv"
)

const version = "0.1"

type Server struct {
    ListenAddress string
    ListenPort int
    MetricsPath string
    ErrorChan chan error
    Handler http.Handler
}

func (srv Server) Start() error {
    // check listenAddress
    if srv.ListenAddress == "localhost" {
	srv.ListenAddress = "127.0.0.1"
    } else if srv.ListenAddress == "any" || srv.ListenAddress == "*" {
	srv.ListenAddress = ""
    }
    if net.ParseIP(srv.ListenAddress) == nil && srv.ListenAddress != "" {
        return fmt.Errorf("Bad value of Server.ListenAddress=%s",srv.ListenAddress)
    }

    // check listenPort
    if srv.ListenPort < 0 || srv.ListenPort > 65535 {
        return fmt.Errorf("Bad value of ListenPort=%d",srv.ListenPort)
    }

    // check metricsPath
    if m, _ := regexp.MatchString("^/[a-zA-Z-_0-9/]*$", srv.MetricsPath); !m {
        return fmt.Errorf("Bad value of MetricsPath=%s",srv.MetricsPath)
    }

    router := http.NewServeMux()
    router.HandleFunc("/",func(w http.ResponseWriter, r *http.Request){
        fmt.Fprintf(w, "<html><head><title>php-fpm exporter</title></head><body><h1>php-fpm exporter</h1><p><a href=\"%s\">%[1]s</a></p></body></html>", srv.MetricsPath)
    })

    router.Handle(srv.MetricsPath,srv.Handler)

    serverAddr := srv.ListenAddress+":"+strconv.Itoa(srv.ListenPort)
    httpServer := &http.Server {
        Addr: serverAddr,
        Handler: router,
    }

    go func() {
        sigint := make(chan os.Signal, 1)
        signal.Notify(sigint, os.Interrupt)
        <-sigint
        if err := httpServer.Shutdown(context.Background()); err != nil && err != http.ErrServerClosed {
            srv.ErrorChan<-err
        }
        close(srv.ErrorChan)
    }()

    go func() {
        if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
            srv.ErrorChan<-err
        }
        close(srv.ErrorChan)
    }()

    return nil
}

func (srv Server) handlerMetrics(w http.ResponseWriter, r *http.Request) {
//    debugLog.Printf("incoming request from %s: %s \"%s %s%s\"", r.RemoteAddr, r.Proto, r.Method, r.Host, r.RequestURI)
    fmt.Fprintf(w, "metrics")
}
