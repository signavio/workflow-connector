package sql

import (
	"database/sql/driver"
	"fmt"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
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

	log.When(config.Options.Logging).Infoln("[handler -> template] interpolate query string")
	queryString, err := handler.interpolateQueryTemplate()
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- template]\n%s\n", queryString)

	log.When(config.Options.Logging).Infoln("[handler -> db] get query results")
	results, err := b.queryContext(req.Context(), queryString, id)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%#v\n",
		results,
	)
	if len(results) == 0 {
		rw.WriteHeader(http.StatusNotFound)
		return
	}
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

// TestCases
var TestCasesGetSingle = []TestCase{
	{
		Kind:             "success",
		Name:             "it succeeds when a table contains more than one column",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L),999,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 999,
    "currency": "EUR"
  },
  "id": "1",
  "name": "Stainless Steel Mash Tun (50L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("1").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/1", nil)
			return req
		}(),
	},
	{

		Kind:             "failure",
		Name:             "it fails and returns 404 NOT FOUND when querying a non existent id",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv:       "",
		ExpectedResults: ``,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment/42", nil)
			return req
		}(),
	},
}
