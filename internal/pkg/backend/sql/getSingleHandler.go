package sql

import (
	"fmt"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetSingle(rw http.ResponseWriter, req *http.Request) {
	id := req.Context().Value(util.ContextKey("id")).(string)
	routeName := req.Context().Value(util.ContextKey("currentRoute")).(string)
	table := req.Context().Value(util.ContextKey("table")).(string)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	queryTemplate := b.Templates[routeName]
	relations := req.Context().Value(util.ContextKey("relationships")).([]*config.Field)
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			Relations      []*config.Field
			UniqueIDColumn string
		}{
			TableName:      table,
			Relations:      relations,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler] interpolate query string")
	queryString, err := handler.interpolateQueryTemplate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infoln(queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.queryContext(req.Context(), queryString, id)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(results) == 0 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
	// deduplicate the results fetched when querying the database
	results = deduplicateSingleResource(
		results,
		util.GetTypeDescriptorUsingDBTableName(
			config.Options.Descriptor.TypeDescriptors,
			table,
		),
	)
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n", results)
	log.When(config.Options.Logging).Infoln("[handler -> formatter] format results as json")
	formattedResults, err := formatting.WorkflowAccelerator.Format(req, results)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- formatter] formatted results: \n%s\n",
		formattedResults,
	)
	isCreated, ok := req.Context().Value(util.ContextKey("isCreated")).(bool)
	if ok && isCreated {
		rw.Header().Set("Location", fmt.Sprintf("%s/%s/%s", req.Host, table, id))
		rw.WriteHeader(http.StatusCreated)
		rw.Write(formattedResults)
	} else {
		rw.Write(formattedResults)
	}
	return
}
