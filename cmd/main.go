package main

import (
	"net/http"

	"github.com/ucl-arc-tre/egress/internal/config"
	"github.com/ucl-arc-tre/egress/internal/handler"
	"github.com/ucl-arc-tre/egress/internal/middleware"
	"github.com/ucl-arc-tre/egress/internal/openapi"
	"github.com/ucl-arc-tre/egress/internal/router"

	"github.com/ucl-arc-tre/x/pkg/graceful"
)

func main() {
	config.Init()

	handler := handler.New()
	router := router.New(handler)
	openapi.RegisterHandlersWithOptions(router, handler,
		openapi.GinServerOptions{
			BaseURL:     config.BaseURL,
			Middlewares: middleware.All(),
		})

	server := &http.Server{
		Addr:              config.ServerAddress(),
		Handler:           router.Handler(),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}
	graceful.Serve(server, config.ServerShutdownDuration)
}
