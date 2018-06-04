package sql

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) DeleteSingle(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	requestTx := mux.Vars(req)["tx"]
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	queryTemplate := b.Templates[routeName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			UniqueIDColumn string
		}{
			TableName:      table,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> template] interpolate query string")
	queryString, err := handler.interpolateQueryTemplate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- template]\n%s\n", queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")

	// Check that user provided tx is already in backend.Transactions
	if requestTx != "" {
		tx, ok := b.Transactions.Load(requestTx)
		if !ok {
			msg := &util.HTTPErrorCodeMsg{
				http.StatusNotFound,
				fmt.Sprintf(
					"Transaction with uuid '%s' not found in %s backend",
					requestTx, table,
				),
			}
			http.Error(rw, msg.String(), http.StatusNotFound)
			return
		}
		log.When(config.Options.Logging).Infof("Query will execute within user specified transaction:\n%s\n", tx)
	}
	result, err := b.execContext(req.Context(), queryString, id)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%#v\n",
		result,
	)
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		rw.WriteHeader(http.StatusNotFound)
		msg := &util.HTTPErrorCodeMsg{
			http.StatusNotFound,
			fmt.Sprintf(
				"Resource with uniqueID '%s' not found in %s table",
				id, table,
			),
		}
		rw.Write([]byte(msg.String()))
		return
	}
	msg := &util.HTTPCodeMsg{
		http.StatusOK,
		fmt.Sprintf(
			"Resource with uniqueID '%s' successfully deleted from %s table",
			id, table,
		),
	}
	rw.Write([]byte(msg.String()))
	return
}
