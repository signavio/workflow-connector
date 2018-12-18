package middleware

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

// RouteChecker checks to make sure that the type descriptor key provided
// by the user actually matches to a database table name
func RouteChecker(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The value stored in the {table} variable is acutally the "key"
		// property of the type descriptor in the descriptor.json file
		// and should match with a table name in the database
		typeDescriptorKey := mux.Vars(r)["table"]
		if len(typeDescriptorKey) == 0 {
			next.ServeHTTP(w, r)
			return
		}
		_, ok := util.GetDBTableNameUsingTypeDescriptorKey(
			config.Options.Descriptor.TypeDescriptors,
			typeDescriptorKey,
		)
		if !ok {
			msg := &util.ResponseMessage{
				Code: http.StatusNotFound,
				Msg: fmt.Sprintf(
					"The requested handler '%s' does not exist",
					typeDescriptorKey,
				),
			}
			http.Error(w, msg.String(), http.StatusNotFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}
