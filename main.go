package main

import (
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/app/server"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
)

var version string

func main() {
	log.When(true).Infof("starting workflow connector v%s\n", version)
	endpoint, err := endpoint.NewEndpoint(config.Options)
	if err != nil {
		log.Fatalln(err)
	}
	log.When(true).Infoln("[endpoint] initialize backend")
	err = endpoint.Open(
		config.Options.Database.Driver,
		config.Options.Database.URL,
	)
	if err != nil {
		log.Fatalln(err)
	}
	server := server.NewServer(config.Options, endpoint)
	println("[server] ready and listening on :" + config.Options.Port)
	if config.Options.TLS.Enabled {
		log.Fatalln(server.ListenAndServeTLS(config.Options.TLS.PublicKey,
			config.Options.TLS.PrivateKey))
	} else {
		log.Fatalln(server.ListenAndServe())
	}
}
