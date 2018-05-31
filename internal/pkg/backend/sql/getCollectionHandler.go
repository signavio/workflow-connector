package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/formatting"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetCollection(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryTemplate := b.Templates[routeName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName string
		}{
			TableName: table,
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
	results, err := b.queryContext(req.Context(), queryString)
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

var TestCasesGetCollection = []TestCase{
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
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L),999,2017-12-12T12:00:00Z\n" +
			"2,HolzbierFaß (200L),512.23,2017-12-12T12:00:00Z\n" +
			"3,Refractometer,129,2017-12-12T12:00:00Z",
		ExpectedResults: `[
  {
    "acquisitionCost": {
      "amount": 999,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Stainless Steel Mash Tun (50L)",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 512.23,
      "currency": "EUR"
    },
    "id": "2",
    "name": "HolzbierFaß (200L)",
    "purchaseDate": "2017-12-12T12:00:00Z"
  },
  {
    "acquisitionCost": {
      "amount": 129,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Refractometer",
    "purchaseDate": "2017-12-12T12:00:00Z"
  }
]`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+)").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment", nil)
			return req
		}(),
	},
}
