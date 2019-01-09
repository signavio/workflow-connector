package backend

import (
	"fmt"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetSingle(rw http.ResponseWriter, req *http.Request) {
	id := req.Context().Value(util.ContextKey("id")).(string)
	routeName := req.Context().Value(util.ContextKey("currentRoute")).(string)
	table := req.Context().Value(util.ContextKey("table")).(string)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	queryUninterpolated := b.GetQueryTemplate(routeName)
	relations := req.Context().Value(util.ContextKey("relationships")).([]*descriptor.Field)
	queryTemplate := &query.QueryTemplate{
		Vars: []string{queryUninterpolated},
		TemplateData: struct {
			TableName      string
			Relations      []*descriptor.Field
			UniqueIDColumn string
		}{
			TableName:      table,
			Relations:      relations,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler] interpolate query string")
	queryString, err := queryTemplate.Interpolate()
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infoln(queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.QueryContext(req.Context(), queryString, id)
	if err != nil {
		switch err.(type) {
		case *util.ResponseMessage:
			http.Error(rw, err.Error(), err.(*util.ResponseMessage).Code)
			return
		default:
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	if len(results) == 0 {
		msg := &util.ResponseMessage{
			Code: http.StatusNotFound,
			Msg:  "requested resource not found",
		}
		rw.WriteHeader(http.StatusNotFound)
		rw.Write(msg.Byte())
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n", results)
	log.When(config.Options.Logging).Infoln("[handler -> formatter] format results as json")
	formattedResults, err := formatting.Standard.Format(req, results)
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
	isCreated, ok := req.Context().Value(util.ContextKey("isCreated")).(bool)
	if ok && isCreated {
		rw.Header().Set(
			"Location",
			fmt.Sprintf("http://%s%s/%s", req.Host, req.URL, id),
		)
		rw.WriteHeader(http.StatusCreated)
		rw.Write(formattedResults)
	} else {
		rw.Write(formattedResults)
	}
	return
}
