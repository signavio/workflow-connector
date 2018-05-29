package server

import (
	"crypto/tls"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/middleware"
	"github.com/urfave/negroni"
)

func NewServer(cfg config.Config, e endpoint.Endpoint) *http.Server {
	router := e.GetHandler().(*mux.Router)
	var server *http.Server
	if cfg.TLS.Enabled {
		server = HTTPServerWithSecureTLSOptions()
	} else {
		server = &http.Server{}
	}
	// Wrap our http Handler functions (i.e. routes) with useful middleware
	// TODO this is cheesy that we are using negroni only for its
	// built in NewRecovery and NewLogger middlewares
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	router.Use(middleware.BasicAuth)
	router.Use(middleware.RequestInjector)
	router.Use(middleware.ResponseInjector)
	n.UseHandler(router)
	server.Addr = ":" + cfg.Port
	server.Handler = n
	return server
}

// HTTPServerWithSecureTLSOptions returns a http server configured to use
// secure cipher suites and curves as defined by the german federal office
// for information security (BSI) in TR-02102-2 version 2018-01
func HTTPServerWithSecureTLSOptions() *http.Server {
	cfg := &tls.Config{
		MinVersion: tls.VersionTLS12,
		CurvePreferences: []tls.CurveID{tls.CurveP521,
			tls.CurveP384, tls.CurveP256},
		PreferServerCipherSuites: true,
		CipherSuites: []uint16{
			tls.TLS_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}
	return &http.Server{
		TLSConfig: cfg,
		TLSNextProto: make(map[string]func(*http.Server,
			*tls.Conn, http.Handler), 0),
	}
}
