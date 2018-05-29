package main

import (
	"log"

	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/app/server"
	"github.com/signavio/workflow-connector/internal/pkg/config"
)

func main() {
	endpoint, err := endpoint.NewEndpoint(config.Options)
	if err != nil {
		panic(err)
	}
	endpoint.Open(
		config.Options.Database.Driver,
		config.Options.Database.URL,
	)
	server := server.NewServer(config.Options, endpoint)
	println("Listening on :" + config.Options.Port)
	if config.Options.TLS.Enabled {
		log.Fatal(server.ListenAndServeTLS(config.Options.TLS.PublicKey,
			config.Options.TLS.PrivateKey))
	} else {
		log.Fatal(server.ListenAndServe())
	}
}
