package server

import (
	"reflect"
	"testing"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/app/endpoint"
	"github.com/signavio/workflow-connector/internal/pkg/config"
)

func TestServerHandlerHasBackendRoutes(t *testing.T) {
	endpoint, _ := endpoint.NewEndpoint(config.Options)
	server := NewServer(config.Options, endpoint)
	routeName := server.Handler.(*mux.Router).
		GetRoute("GetCollectionAsOptionsFilterable").GetName()
	routeHandler := server.Handler.(*mux.Router).
		GetRoute("GetCollectionAsOptionsFilterable").GetHandler()
	routeMethods, _ := server.Handler.(*mux.Router).
		GetRoute("GetCollectionAsOptionsFilterable").GetMethods()
	routePath, _ := server.Handler.(*mux.Router).
		GetRoute("GetCollectionAsOptionsFilterable").GetPathTemplate()
	routeQueries, _ := server.Handler.(*mux.Router).
		GetRoute("GetCollectionAsOptionsFilterable").GetQueriesTemplates()
	if routeName != "GetCollectionAsOptionsFilterable" {
		t.Errorf("Unexpected route name: %s", routeName)
	}
	if routeHandler == nil {
		t.Errorf("Unexpected route handler: %+v", routeHandler)
	}
	if routePath != "/{table}/options" {
		t.Errorf("Unexpected route path: %s", routePath)
	}
	if !reflect.DeepEqual(routeQueries, []string{"filter={filter}"}) {
		t.Errorf("Unexpected route queries: %s", routeQueries)
	}
	if !reflect.DeepEqual(routeMethods, []string{"GET"}) {
		t.Errorf("Unexpected route methods: %s", routeMethods)
	}
}
