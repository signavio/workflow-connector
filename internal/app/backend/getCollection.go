package backend

import (
	//	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetCollection(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryUninterpolated := b.GetQueryTemplate(routeName)
	relations := req.Context().Value(util.ContextKey("relationships")).([]*descriptor.Field)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	requestData, err := util.ParseDataForm(req)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler] requestData: \n%s", requestData)
	columnNames := util.GetColumnNamesFromRequestData(table, requestData)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryUninterpolated},
		TemplateData: struct {
			TableName      string
			Relations      []*descriptor.Field
			UniqueIdColumn string
			ColumnNames []string
		}{
			TableName:      table,
			Relations:      relations,
			UniqueIdColumn: uniqueIDColumn,
			ColumnNames: columnNames,
		},
		CoerceArgFuncs: b.GetCoerceArgFuncs(),
	}

	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler] interpolate query string")
	queryString, args, err := queryTemplate.Interpolate(req.Context(), requestData)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}

	log.When(config.Options.Logging).Infoln(queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.QueryContext(req.Context(), queryString, args...)
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

	log.When(config.Options.Logging).Infoln("[handler -> Format] format results as json")
	formattedResults, err := formatting.Collection.Format(req.Context(), results)
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
