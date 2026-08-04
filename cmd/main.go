package main

import (
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/handler"
	"github.com/ucl-arc-tre/egress/internal/middleware"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/router"

	"github.com/ucl-arc-tre/x/pkg/graceful"
)

func main() {
	config.Init()

	if config.MutualTLSEnabled() {
		router := router.New(handler.NewHealth())
		server := &http.Server{
			Addr:              ":" + config.HttpPort(),
			Handler:           router.Handler(),
			ReadHeaderTimeout: 1 * time.Second,
		}
		log.Info().Str("addr", server.Addr).Msg("Starting health server")
		go graceful.Serve(server, 1*time.Second)
	}

	handler := handler.New()
	router := router.New(handler)
	openapi.RegisterHandlersWithOptions(router, handler,
		openapi.GinServerOptions{
			BaseURL:     config.BaseURL,
			Middlewares: middleware.All(),
		},
	)

	server := &http.Server{
		Addr:              config.ServerAddress(),
		Handler:           router.Handler(),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}
	if config.MutualTLSEnabled() {
		server.TLSConfig = config.MutualTLS()
		graceful.ServeTLS(server, config.ServerShutdownDuration)
	} else {
		graceful.Serve(server, config.ServerShutdownDuration)
	}
}
