package sql

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	supportedOperators = []struct {
		label         string
		sqlEquivalent string
	}{
		{
			label:         "eq",
			sqlEquivalent: "=",
		},
	}
)

type supportedOperator struct {
	label         string
	sqlEquivalent string
}

func (b *Backend) GetCollectionFilterable(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryTemplate := b.Templates[routeName]
	filter := mux.Vars(req)["filter"]
	columnName, err := extractColumnNameFromFilterQueryParam(
		filter,
		util.GetTypeDescriptorUsingDBTableName(
			config.Options.Descriptor.TypeDescriptors,
			table,
		),
	)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	operator, err := extractOperatorFromFilterQueryParam(filter)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	value, err := extractValueFromFilterQueryParam(filter)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			FilterOnColumn string
			Operator       string
		}{
			TableName:      table,
			FilterOnColumn: columnName,
			Operator:       operator,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler] interpolate query string")
	queryString, err := handler.interpolateQueryTemplate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}

	log.When(config.Options.Logging).Infof(
		"[handler -> db] sending the following to db\nquery string: %#v\nwith parameter: %#v\n",
		queryString,
		value,
	)
	results, err := b.queryContext(req.Context(), queryString, value)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%#v\n",
		results,
	)

	log.When(config.Options.Logging).Infoln("[handler -> formatter] format results as json")
	formattedResults, err := formatting.WorkflowAccelerator.Format(req, results)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- formatter] formatted results: \n%s\n",
		formattedResults,
	)

	rw.Write(formattedResults)
	return
}

func extractColumnNameFromFilterQueryParam(filter string, td *config.TypeDescriptor) (columnName string, err error) {
	queryString, err := url.QueryUnescape(filter)
	if err != nil {
		return "", fmt.Errorf("error unescaping query parameter: %s", err)
	}
	parts := strings.Split(queryString, " ")
	for _, field := range td.Fields {
		if parts[0] == field.Key {
			columnName = field.FromColumn
			return columnName, nil
		}
	}
	return "", fmt.Errorf("column '%s' does not exist", parts[0])
}

func extractOperatorFromFilterQueryParam(filter string) (operator string, err error) {
	queryString, err := url.QueryUnescape(filter)
	if err != nil {
		return "", fmt.Errorf("error unescaping query parameter: %s", err)
	}
	parts := strings.Split(queryString, " ")
	for _, op := range supportedOperators {
		if parts[1] == op.label {
			operator = op.sqlEquivalent
			return operator, nil
		}
	}
	return "", fmt.Errorf("operator '%s' is not supported", parts[1])
}

func extractValueFromFilterQueryParam(filter string) (value string, err error) {
	queryString, err := url.QueryUnescape(filter)
	if err != nil {
		return "", fmt.Errorf("error unescaping query parameter: %s", err)
	}
	value = everythingAfterTheOperator(strings.Split(queryString, " "))
	return
}
func everythingAfterTheOperator(queryStringSplitted []string) string {
	return strings.Join(queryStringSplitted[2:], " ")
}
