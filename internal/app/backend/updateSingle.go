package backend

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) UpdateSingle(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryTemplateUninterpolated := b.GetQueryTemplate(routeName)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	requestData, err := util.ParseDataForm(req)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}
	columnNames := getColumnNamesFromRequestData(table, requestData)
	if len(columnNames) == 0 {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg: fmt.Sprintf(
				"the request data contains *one or more* fields "+
					"that are not present in the database\n"+
					"request data:\n%v\n"+
					"fields available in database table:\n%v\n",
				requestData, b.GetSchemaMapping(table).FieldNames),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryTemplateUninterpolated},
		TemplateData: struct {
			TableName      string
			ColumnNames    []string
			UniqueIDColumn string
		}{
			TableName:      table,
			ColumnNames:    columnNames,
			UniqueIDColumn: uniqueIDColumn,
		},
		ColumnNames:        columnNames,
		CoerceExecArgsFunc: b.GetCoerceExecArgsFunc(),
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> query] interpolate query string")
	queryString, args, err := queryTemplate.Interpolate(req.Context(), requestData)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}

	log.When(config.Options.Logging).Infof(
		"[handler -> db] get query results using\nquery string:\n%s"+
			"\nwith the following args:\n%s\n",
		queryString,
		append(args, id),
	)
	result, err := b.ExecContext(req.Context(), queryString, append(args, id)...)
	if err == sql.ErrNoRows {
		msg := &util.ResponseMessage{
			Code: http.StatusNotFound,
			Msg: fmt.Sprintf(
				"Resource with uniqueID '%s' not found in %s table",
				id, table,
			),
		}
		http.Error(rw, msg.Error(), http.StatusNotFound)
		return
	}
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
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
