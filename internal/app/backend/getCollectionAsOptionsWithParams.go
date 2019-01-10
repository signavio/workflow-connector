package backend

import (
	"fmt"
	"net/http"
	"net/url"

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
	paramsWithValues := mapQueryParameterNamesToColumnNames(table, req.URL.Query())
	filter := fmt.Sprintf("%%%s%%", mux.Vars(req)["filter"])
	queryUninterpolated := b.GetQueryTemplate(routeName)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryUninterpolated},
		TemplateData: struct {
			TableName          string
			UniqueIDColumn     string
			ParamsWithValues   map[string]string
			ColumnAsOptionName string
		}{
			TableName:          table,
			UniqueIDColumn:     uniqueIDColumn,
			ParamsWithValues:   paramsWithValues,
			ColumnAsOptionName: columnAsOptionName,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> backend] interpolate query string")
	queryString, err := queryTemplate.Interpolate()
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- backend]\n%s\n", queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.QueryContext(req.Context(), queryString, filter)
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

func mapQueryParameterNamesToColumnNames(tableName string, u url.Values) (paramsWithValues map[string]string) {
	paramsWithValues = make(map[string]string)
	values := urlValuesWithoutFilter(u)
	for k, v := range values {
		columnName, ok := util.GetColumnNameFromQueryParameterName(
			config.Options.Descriptor.TypeDescriptors,
			tableName,
			k,
		)
		if ok {
			paramsWithValues[columnName] = v[0]
		}
	}
	return
}
func urlValuesWithoutFilter(u url.Values) url.Values {
	if len(u["filter"]) > 1 {
		// There exists a type descriptor with a field whose key name
		// is  literaly 'filter', assume the second occurence of
		// 'filter' is the actual parameter upon which we
		// want to prefilter the result set
		val := u["filter"][1]
		u.Del("filter")
		u.Add("filter", val)
		return u
	}
	return u
}
