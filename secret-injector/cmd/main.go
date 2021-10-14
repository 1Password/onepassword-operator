package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/1Password/onepassword-operator/secret-injector/pkg/webhook"
	"github.com/golang/glog"
)

const (
	connectTokenSecretKeyEnv  = "OP_CONNECT_TOKEN_KEY"
	connectTokenSecretNameEnv = "OP_CONNECT_TOKEN_NAME"
	connectHostEnv            = "OP_CONNECT_HOST"
)

func main() {
	var parameters webhook.WebhookServerParameters

	glog.Info("Starting webhook")
	// get command line parameters
	flag.IntVar(&parameters.Port, "port", 8443, "Webhook server port.")
	flag.StringVar(&parameters.CertFile, "tlsCertFile", "/etc/webhook/certs/cert.pem", "File containing the x509 Certificate for HTTPS.")
	flag.StringVar(&parameters.KeyFile, "tlsKeyFile", "/etc/webhook/certs/key.pem", "File containing the x509 private key to --tlsCertFile.")
	flag.Parse()

	pair, err := tls.LoadX509KeyPair(parameters.CertFile, parameters.KeyFile)
	if err != nil {
		glog.Errorf("Failed to load key pair: %v", err)
	}

	connectHost, present := os.LookupEnv(connectHostEnv)
	if !present {
		glog.Error("")
	}

	connectTokenName, present := os.LookupEnv(connectTokenSecretNameEnv)
	if !present {
		glog.Error("")
	}

	connectTokenKey, present := os.LookupEnv(connectTokenSecretKeyEnv)
	if !present {
		glog.Error("")
	}

	webhookConfig := webhook.Config{
		ConnectHost:      connectHost,
		ConnectTokenName: connectTokenName,
		ConnectTokenKey:  connectTokenKey,
	}
	webhookServer := &webhook.WebhookServer{
		Config: webhookConfig,
		Server: &http.Server{
			Addr:      fmt.Sprintf(":%v", parameters.Port),
			TLSConfig: &tls.Config{Certificates: []tls.Certificate{pair}},
		},
	}

	// define http server and server handler
	mux := http.NewServeMux()
	mux.HandleFunc("/inject", webhookServer.Serve)
	webhookServer.Server.Handler = mux

	// start webhook server in new rountine
	go func() {
		if err := webhookServer.Server.ListenAndServeTLS("", ""); err != nil {
			glog.Errorf("Failed to listen and serve webhook server: %v", err)
		}
	}()

	// listening OS shutdown singal
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan

	glog.Infof("Got OS shutdown signal, shutting down webhook server gracefully...")
	webhookServer.Server.Shutdown(context.Background())
}
