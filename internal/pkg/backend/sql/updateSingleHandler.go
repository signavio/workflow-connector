package sql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var TestCasesUpdateSingle = []TestCase{
	{
		Kind:             "success",
		Name:             "it succeeds when provided with valid parameters as URL parameters",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv: "2,HolzbierFaß (100L),299.99,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 299.99,
    "currency": "EUR"
  },
  "id": "2",
  "name": "HolzbierFaß (100L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (100L)", "299.99", "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("name", "HolzbierFaß (100L)")
			postData.Set("acquisitionCost", "299.99")
			req, _ := http.NewRequest("PATCH", "/equipment/2?"+postData.Encode(), nil)
			return req
		}(),
	},
	{
		Kind:             "success",
		Name:             "it succeeds when provided with valid parameters as json in the request body",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv: "2,HolzbierFaß (200L),512.23,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 512.23,
    "currency": "EUR"
  },
  "id": "2",
  "name": "HolzbierFaß (200L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (200L)", 512.23, "2").
				WillReturnResult(sqlmock.NewResult(2, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("2").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest(
				"PATCH",
				"/equipment/2",
				strings.NewReader(
					"{\"name\": \"HolzbierFaß (200L)\","+
						"\"acquisitionCost\": 512.23}"),
			)
			req.Header = map[string][]string{
				"Content-Type": []string{"application/json"},
			}
			return req
		}(),
	},
	{

		Kind:             "failure",
		Name:             "it fails and returns 404 NOT FOUND when trying to update a non existent id",
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
			mock.ExpectBegin()
			mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
				WithArgs("HolzbierFaß (200L)", "512.23", "42").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("name", "HolzbierFaß (200L)")
			postData.Set("acquisitionCost", "512.23")
			req, _ := http.NewRequest("PATCH", "/equipment/42?"+postData.Encode(), nil)
			return req
		}(),
	},
}

func (b *Backend) UpdateSingle(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := mux.Vars(req)["table"]
	requestTx := mux.Vars(req)["tx"]
	queryTemplate := b.Templates[routeName]
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	requestData, err := util.ParseDataForm(req)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusBadRequest)
		return
	}
	columnNames := getColumnNamesFromRequestData(table, requestData)
	if len(columnNames) == 0 {
		http.Error(rw, fmt.Sprintf(
			"the request data contains *one or more* fields "+
				"that are not present in the database\n"+
				"request data:\n%v\n"+
				"fields available in database table:\n%v\n",
			requestData, b.TableSchemas[table].ColumnNames),
			http.StatusBadRequest,
		)
		return
	}
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			ColumnNames    []string
			UniqueIDColumn string
		}{
			TableName:      table,
			ColumnNames:    columnNames,
			UniqueIDColumn: uniqueIDColumn,
		},
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> template] interpolate query string")
	queryString, args, err := handler.interpolateExecTemplates(req.Context(), requestData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- template]\n%s\n", queryString)
	log.When(config.Options.Logging).Infof("will be called with these args:\n%s\n", args)

	// Check that user provided tx is already in backend.Transactions
	if requestTx != "" {
		tx, ok := b.Transactions.Load(requestTx)
		if !ok {
			http.Error(rw, string(failureMsg(requestTx, table)[:]), http.StatusNotFound)
			return
		}
		log.When(config.Options.Logging).Infof("Query will execute within user specified transaction:\n%s\n", tx)
	}
	result, err := b.execContext(req.Context(), queryString, args...)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%#v\n", result)

	withUpdatedRoute := context.WithValue(
		req.Context(),
		util.ContextKey("currentRoute"),
		"GetSingle",
	)
	newReq := req.WithContext(withUpdatedRoute)
	b.GetSingle(rw, newReq)
	return
}
