package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/formatting"
	"github.com/signavio/workflow-connector/pkg/log"
	"github.com/signavio/workflow-connector/pkg/util"
	"github.com/urfave/negroni"
)

// App stores user config, a connection to and endpoint, a http.Server,
// and a gorilla/mux router
type App struct {
	Cfg *config.Config
	Endpoint
	Server    *http.Server
	Router    *mux.Router
	Formatter formatting.JSONForWfa
}

// NewApp returns an instance of App which has loaded user config,
// created a connection to an endpoint and configured a http.Server
// which listens on typical routes defined by a standard REST API
func NewApp(cfg *config.Config) *App {
	router := mux.NewRouter()
	endpoint, err := NewEndpoint(cfg, router)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	var server *http.Server
	if cfg.TLS.Enabled {
		server = config.HTTPServerWithSecureTLSOptions()
	} else {
		server = &http.Server{}
	}
	authMiddleware := config.BasicAuth(cfg)
	// Wrap our http Handler functions (routes) with useful middleware
	n := negroni.New(negroni.NewRecovery(), negroni.NewLogger())
	n.Use(negroni.HandlerFunc(authMiddleware))
	n.UseHandler(router)
	server.Addr = ":" + cfg.Port
	server.Handler = n
	app := &App{cfg, endpoint, server, router, formatting.JSONForWfa{}}
	return app
}

func (app *App) DefineRoutes() {
	app.Router.HandleFunc("/{table}/options/{id}", app.getSingleAsOption).
		Methods("GET")
	app.Router.HandleFunc("/{table}/options", app.getCollectionAsOptionsFilterable).
		Methods("GET").Queries("filter", "{filter}")
	app.Router.HandleFunc("/{table}/options", app.getCollectionAsOptions).
		Methods("GET")
	app.Router.HandleFunc("/{table}/{id}", app.getSingle).
		Methods("GET")
	app.Router.HandleFunc("/{table}/{id}", app.updateSingle).
		Methods("PUT")
	app.Router.HandleFunc("/{table}", app.getCollection).
		Methods("GET")
	app.Router.HandleFunc("/{table}", app.createSingle).
		Methods("POST")
	app.Router.HandleFunc("/", app.getDescriptorFile).
		Methods("GET")
	// TODO
	// app.Router = appendEndpointSpecificRoutes()
}
func (app *App) getDescriptorFile(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> http.ServeFile] descriptor file")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getDescriptorFile"),
	)
	http.ServeFile(rw, requestWithActiveRoute, "./config/descriptor.json")
}

func (app *App) getSingle(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] getSingle")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getSingle"),
	)
	app.commonRouteHandler(app.Endpoint.GetSingle, rw, requestWithActiveRoute)
}

func (app *App) getCollection(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] getCollection")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getCollection"),
	)
	app.commonRouteHandler(app.Endpoint.GetCollection, rw, requestWithActiveRoute)
}

func (app *App) getSingleAsOption(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] getSingleAsOption")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getSingleAsOption"),
	)
	app.commonRouteHandler(app.Endpoint.GetSingleAsOption, rw, requestWithActiveRoute)
}

func (app *App) getCollectionAsOptions(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] getCollectionAsOptions")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getCollectionAsOptions"),
	)
	app.commonRouteHandler(app.Endpoint.GetCollectionAsOptions, rw, requestWithActiveRoute)
}

func (app *App) getCollectionAsOptionsFilterable(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] getCollectionAsOptionsFilterable")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"getCollectionAsOptionsFilterable"),
	)
	app.commonRouteHandler(app.Endpoint.GetCollectionAsOptionsFilterable, rw, requestWithActiveRoute)
}

func (app *App) updateSingle(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] updateSingle")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"updateSingle"),
	)
	app.commonRouteHandler(app.Endpoint.UpdateSingle, rw, requestWithActiveRoute)
}

func (app *App) createSingle(rw http.ResponseWriter, req *http.Request) {
	log.When(app.Cfg).Infoln("[request -> routeHandler] createSingle")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			config.ContextKey("route"),
			"createSingle"),
	)
	app.commonRouteHandler(app.Endpoint.CreateSingle, rw, requestWithActiveRoute)
}

func (app *App) commonRouteHandler(method func(*http.Request) ([]interface{}, error), rw http.ResponseWriter, req *http.Request) {
	tableFromRequest := mux.Vars(req)["table"]
	if !contains(app.Cfg.Descriptor.TypeDescriptors, tableFromRequest) {
		http.Error(rw, fmt.Sprintf("The resource, `%v` that you requested is not"+
			" in your descriptor.json file. Please define it there first.",
			tableFromRequest), http.StatusNotFound)
		return
	}
	request, cancel := util.BuildRequest(req, app.Cfg.Descriptor.TypeDescriptors, tableFromRequest)
	defer cancel()
	log.When(app.Cfg).Infoln("[routeHandler -> handlers]")
	results, err := method(request)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(app.Cfg).Infoln("[routeHandler <- handlers]")
	log.When(app.Cfg).Infoln("[routeHandler -> formatter]")
	formattedResults, err := app.Formatter.Format(request.Context(), app.Cfg, results)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	rw.Header().Set("Content-Type", "application/json")
	if app.Cfg.TLS.Enabled {
		rw.Header().Add("Strict-Transport-Security",
			"max-age=63072000; includeSubDomains")
	}
	rw.Write(formattedResults)
	return
}

func contains(typeDescriptors []*config.TypeDescriptor, table string) (result bool) {
	result = false
	for _, v := range typeDescriptors {
		if v.Key == table {
			result = true
		}
	}
	return result
}
