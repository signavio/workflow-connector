package backend

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) DeleteSingle(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	queryUninterpolated := b.GetQueryTemplate(routeName)
	queryTemplate := &query.QueryTemplate{Vars: []string{queryUninterpolated}, TemplateData: struct {
		TableName      string
		UniqueIDColumn string
	}{
		TableName:      table,
		UniqueIDColumn: uniqueIDColumn,
	},
		CoerceArgFuncs: b.GetCoerceArgFuncs(),
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> backend] interpolate query string")
	queryString, _, err := queryTemplate.Interpolate(req.Context(), nil)
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
	result, err := b.ExecContext(req.Context(), queryString, id)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n",
		result,
	)
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
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
	msg := &util.ResponseMessage{
		Code: http.StatusOK,
		Msg: fmt.Sprintf(
			"Resource with uniqueID '%s' successfully deleted from %s table",
			id, table,
		),
	}
	rw.Write(msg.Byte())
	return
}
