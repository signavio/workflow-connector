package backend

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/filter"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetCollectionFilterable(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryUninterpolated := b.GetQueryTemplate(routeName)
	relations := req.Context().Value(util.ContextKey("relationships")).([]*descriptor.Field)
	filterExpression, err := filter.New(req.Context(), mux.Vars(req)["filter"])
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryUninterpolated},
		TemplateData: struct {
			TableName      string
			Relations      []*descriptor.Field
			FilterOnColumn string
			Operator       string
		}{
			TableName:      table,
			Relations:      relations,
			FilterOnColumn: string(filterExpression.Arguments[0]),
			Operator:       b.GetFilterPredicateMapping(filterExpression.Predicate),
		},
		CoerceArgFuncs:   b.GetCoerceArgFuncs(),
		QueryFormatFuncs: b.GetQueryFormatFuncs(),
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler] interpolate query string")
	queryString, _, err := queryTemplate.Interpolate(req.Context(), nil)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}

	log.When(config.Options.Logging).Infof(
		"[handler -> db] sending the following to db\nquery string: %#v\nwith parameter: %#v\n",
		queryString,
		filterExpression.Arguments[1],
	)
	results, err := b.QueryContext(
		req.Context(),
		queryString,
		string(filterExpression.Arguments[1]))
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
	formattedResults, err := formatting.Standard.Format(req.Context(), results)
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
