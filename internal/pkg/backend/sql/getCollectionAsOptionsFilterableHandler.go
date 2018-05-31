package sql

import (
	"database/sql/driver"
	"fmt"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetCollectionAsOptionsFilterable(rw http.ResponseWriter, req *http.Request) {
	log.When(config.Options.Logging).Infoln("[handler] GetSingleAsOption")
	routeName := mux.CurrentRoute(req).GetName()
	table := mux.Vars(req)["table"]
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	columnAsOptionName := req.Context().Value(util.ContextKey("columnAsOptionName")).(string)
	filter := fmt.Sprintf("%%%s%%", mux.Vars(req)["filter"])
	queryTemplate := b.Templates[routeName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName          string
			UniqueIDColumn     string
			ColumnAsOptionName string
		}{
			TableName:          table,
			UniqueIDColumn:     uniqueIDColumn,
			ColumnAsOptionName: columnAsOptionName,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> template] interpolate query string")
	queryString, err := handler.interpolateQueryTemplate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- template]\n%s\n", queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.queryContextForOptionRoutes(req.Context(), queryString, filter)
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

var TestCasesGetCollectionAsOptionsFilterable = []TestCase{
	{
		Kind:             "success",
		Name:             "it succeeds when a table contains more than one column",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
		},
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L)",
		ExpectedResults: `{
  "id": "1",
  "name": "Stainless Steel Mash Tun (50L)"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT (.+), (.+) FROM (.+) WHERE (.+) LIKE (.+)").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/options?filter=stain", nil)
			return req
		}(),
	},
}
