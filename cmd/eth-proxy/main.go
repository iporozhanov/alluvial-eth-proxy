package main

import (
	"eth-proxy/internal/app"
	"eth-proxy/internal/config"
	"eth-proxy/internal/eth_client"
	"eth-proxy/internal/handlers/http"
	"eth-proxy/internal/logger"
	"go.uber.org/zap"
	"os"
	"os/signal"
)

func main() {
	cfg := config.Instance()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	go func() {
		<-sigChan
		os.Exit(0)
	}()

	l, err := logger.New(cfg.LogLevel)
	if err != nil {
		panic(err)
	}

	l.Infof("test!!!!!_______")

	defer func(l *zap.SugaredLogger) {
		_ = l.Sync()
	}(l)

	proxy := eth_client.NewNodeProxy(l)

	for _, url := range cfg.ClientUrls() {
		c, err := eth_client.NewEthClient(url)
		if err != nil {
			l.Errorw("failed to create client", "url", url, "err", err)
		}

		proxy.AddClient(url, c)
	}

	proxy.Start()

	a := app.New(l, proxy)

	h := http.New(a, l)

	if err = h.Start(cfg.HTTPPort, cfg.HTTPTimeoutLimit); err != nil {
		panic(err)
	}

}
