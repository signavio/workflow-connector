package middleware

import (
	"context"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

// RequestInjector will add necessary key value pairs to the context in request
// that will be used later
func RequestInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(mux.Vars(r)["table"]) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		tx := mux.Vars(r)["tx"]
		id := mux.Vars(r)["id"]
		routeName := mux.CurrentRoute(r).GetName()
		// The value stored in the {table} variable is acutally the "key"
		// property of the type descriptor in the descriptor.json file
		// and *not* the name of the table in the database
		typeDescriptorKey := mux.Vars(r)["table"]
		tableName := util.GetDBTableNameUsingTypeDescriptorKey(
			config.Options.Descriptor.TypeDescriptors,
			typeDescriptorKey,
		)
		typeDescriptor := util.GetTypeDescriptorUsingTypeDescriptorKey(
			config.Options.Descriptor.TypeDescriptors,
			typeDescriptorKey,
		)
		withCurrentRoute := context.WithValue(
			r.Context(),
			util.ContextKey("currentRoute"),
			routeName,
		)
		withID := context.WithValue(
			withCurrentRoute,
			util.ContextKey("id"),
			id,
		)
		withTx := context.WithValue(
			withID,
			util.ContextKey("tx"),
			tx,
		)
		withTable := context.WithValue(
			withTx,
			util.ContextKey("table"),
			tableName,
		)
		withColumnAsOptionName := context.WithValue(
			withTable,
			util.ContextKey("columnAsOptionName"),
			typeDescriptor.ColumnAsOptionName,
		)
		withUniqueIDColumn := context.WithValue(
			withColumnAsOptionName,
			util.ContextKey("uniqueIDColumn"),
			typeDescriptor.UniqueIdColumn,
		)
		withRelationships := context.WithValue(
			withUniqueIDColumn,
			util.ContextKey("relationships"),
			util.TypeDescriptorRelationships(typeDescriptor),
		)
		newReq := r.WithContext(withRelationships)
		// TODO
		//contextWithCancel, cancelFn := context.WithCancel(contextWithRelationships)

		//return req.WithContext(contextWithCancel), cancelFn
		next.ServeHTTP(w, newReq)
	})
}

// ResponseInjector will add security parameters to the response header
func ResponseInjector(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if config.Options.TLS.Enabled {
			w.Header().Add("Strict-Transport-Security",
				"max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}
