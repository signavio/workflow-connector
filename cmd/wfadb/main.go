package main

import (
	"log"

	"github.com/signavio/workflow-connector/pkg/app"
	"github.com/signavio/workflow-connector/pkg/config"
)

func main() {
	withDescriptorFile, err := config.LocationsForDescriptorFile()
	if err != nil {
		panic(err)
	}
	cfg := config.Initialize(withDescriptorFile)
	a := app.NewApp(cfg)
	a.DefineRoutes()
	println("Listening in :" + a.Cfg.Port)
	if a.Cfg.TLS.Enabled {
		log.Fatal(a.Server.ListenAndServeTLS(a.Cfg.TLS.PublicKey,
			a.Cfg.TLS.PrivateKey))
	} else {
		log.Fatal(a.Server.ListenAndServe())
	}
}
