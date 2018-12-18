package main

import (
	"context"
	"net/http"
	"os"

	"github.com/kardianos/service"
	"github.com/signavio/workflow-connector/internal/app"
	"github.com/signavio/workflow-connector/internal/app/server"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/spf13/viper"
)

var (
	version string
	logger  service.Logger
)

type App struct {
	server *http.Server
}

func (a *App) Start(s service.Service) error {
	logger.Infof("starting workflow connector %s\n", version)
	go a.run()
	return nil
}
func (a *App) Stop(s service.Service) error {
	logger.Infof("\nstopping workflow connector %s\n", version)
	if err := a.server.Shutdown(context.Background()); err != nil {
		logger.Infof("unable to shutdown server cleanly: %s\n", err)
	}
	if service.Interactive() {
		os.Exit(0)
	}
	return nil
}
func (a *App) run() {
	endpoint, err := app.NewEndpoint(config.Options)
	if err != nil {
		logger.Errorf("unable to create new endpoint: %s\n", err)
		os.Exit(1)
	}
	err = endpoint.Open(
		config.Options.Database.Driver,
		config.Options.Database.URL,
	)
	if err != nil {
		logger.Errorf("unable to initialize backend: %s\n", err)
		os.Exit(1)
	}
	a.server = server.NewServer(config.Options, endpoint)
	logger.Infof(
		"server is ready and listening on port %s\n",
		config.Options.Port,
	)
	if config.Options.TLS.Enabled {
		err := a.server.ListenAndServeTLS(
			config.Options.TLS.PublicKey,
			config.Options.TLS.PrivateKey,
		)
		if err != http.ErrServerClosed {
			logger.Errorf("unable to start http server: %s\n", err)
			os.Exit(1)
		}
	} else {
		err := a.server.ListenAndServe()
		if err != http.ErrServerClosed {
			logger.Errorf("unable to start http server: %s\n", err)
			os.Exit(1)
		}
	}
}

func main() {
	a := &App{}
	svc, err := service.New(
		a,
		&service.Config{
			Name:        config.Options.Name,
			DisplayName: config.Options.DisplayName,
			Description: config.Options.Description,
		},
	)
	if err != nil {
		logger.Errorf("unable to create service: %s\n", err)
		os.Exit(1)
	}
	logger, err = svc.Logger(nil)
	if err != nil {
		logger.Errorf("unable to initialize logger: %s\n", err)
		os.Exit(1)
	}
	serviceControl, ok := viper.Get("service").(string)
	if ok && serviceControl != "" {
		// Execute user specified control on service
		if err := service.Control(svc, serviceControl); err != nil {
			logger.Error(err)
			os.Exit(1)
		}
		return
	}
	err = svc.Run()
	if err != nil {
		logger.Errorf("unable to run the service: %s\n", err)
		os.Exit(1)
	}
}
