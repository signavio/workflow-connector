package sql

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) UpdateSingle(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	requestTx := mux.Vars(req)["tx"]
	queryTemplate := b.Templates[routeName]
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	requestData, err := util.ParseDataForm(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	columnNames := getColumnNamesFromRequestData(table, requestData)
	if len(columnNames) == 0 {
		http.Error(rw, fmt.Sprintf(
			"the request data contains *one or more* fields "+
				"that are not present in the database\n"+
				"request data:\n%v\n"+
				"fields available in database table:\n%v\n",
			requestData, b.TableSchemas[table].ColumnNames),
			http.StatusBadRequest,
		)
		return
	}
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			ColumnNames    []string
			UniqueIDColumn string
		}{
			TableName:      table,
			ColumnNames:    columnNames,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> backend] interpolate query string")
	queryString, args, err := handler.interpolateExecTemplates(req.Context(), requestData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- backend] interpolated query string:\n%s\n", queryString)
	typeDescriptor := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		table,
	)
	queryStringWithFormatting := b.InjectFormattingFuncs(
		queryString,
		columnNames,
		typeDescriptor.Fields,
	)
	log.When(config.Options.Logging).Infof(
		"[handler <- backend] query string with injected formatting functions:\n%s\n",
		queryStringWithFormatting,
	)
	var result sql.Result
	// Check that user provided tx is already in backend.Transactions
	if requestTx != "" {
		tx, ok := b.Transactions.Load(requestTx)
		if !ok {
			msg := &util.HTTPCodeMsg{
				http.StatusNotFound,
				fmt.Sprintf(
					"Transaction with uuid '%s' not found in %s backend",
					requestTx, table,
				),
			}
			http.Error(rw, msg.String(), http.StatusNotFound)
			return
		}
		log.When(config.Options.Logging).Infof("[handler] Query will execute within user specified transaction:\n%s\n", tx)
		result, err = b.TransactWithinTx(req.Context(), tx.(*sql.Tx), queryStringWithFormatting, args...)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		result, err = b.TransactDirectly(req.Context(), b.DB, queryStringWithFormatting, args...)
		if err != nil {
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n", result)

	withUpdatedRoute := context.WithValue(
		req.Context(),
		util.ContextKey("currentRoute"),
		"GetSingle",
	)
	newReq := req.WithContext(withUpdatedRoute)
	b.GetSingle(rw, newReq)
	return
}
