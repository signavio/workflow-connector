package backend

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetCollectionAsOptionsWithParams(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	columnAsOptionName := req.Context().Value(util.ContextKey("columnAsOptionName")).(string)
	var args []interface{}
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
	filter := fmt.Sprintf("%%%s%%", mux.Vars(req)["filter"])
	queryUninterpolated := b.GetQueryTemplate(routeName)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryUninterpolated},
		TemplateData: struct {
			TableName          string
			UniqueIDColumn     string
			ColumnAsOptionName string
			ColumnNames        []string
		}{
			TableName:          table,
			UniqueIDColumn:     uniqueIDColumn,
			ColumnAsOptionName: columnAsOptionName,
			ColumnNames:        columnNames,
		},
		ColumnNames:      columnNames,
		CoerceArgFuncs:   b.GetCoerceArgFuncs(),
		QueryFormatFuncs: b.GetQueryFormatFuncs(),
	}
	log.When(config.Options.Logging).Infof("[handler] %s", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> backend] interpolate query string")
	queryString, args, err := queryTemplate.Interpolate(req.Context(), requestData)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- backend]\n%s\n", queryString)

	log.When(config.Options.Logging).Infof(
		"[handler -> db] get query results using\nquery string:\n%s"+
			"\nwith the following args:\n%s\n",
		queryString,
		append([]interface{}{filter}, args...),
	)
	results, err := b.QueryContext(req.Context(), queryString, append([]interface{}{filter}, args...)...)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n",
		results,
	)

	log.When(config.Options.Logging).Infoln("[handler -> formatter] format results as json")
	formattedResults, err := formatting.GetCollectionAsOptionsFilterable.Format(req, results)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- formatter] formatted results: \n%s\n",
		formattedResults,
	)

	rw.Write(formattedResults)
	return
}
