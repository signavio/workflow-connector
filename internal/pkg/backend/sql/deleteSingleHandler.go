package sql

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var TestCasesDeleteSingle = []TestCase{
	{
		Kind:             "success",
		Name:             "it succeeds in deleting an existing resource",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv: "",
		ExpectedResults: `{
  "status": {
    "code": "200",
    "description": "Resource with uniqueID '4' successfully deleted from equipment table"
  }
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM (.+) WHERE (.+) = (.+)").
				WithArgs("4").
				WillReturnResult(sqlmock.NewResult(4, 1))
			mock.ExpectCommit()
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("DELETE", "/equipment/4", nil)
			return req
		}(),
	},
	{

		Kind:             "failure",
		Name:             "it fails and returns 404 NOT FOUND when trying to delete a non existent id",
		DescriptorFields: commonDescriptorFields,
		TableSchema:      commonTableSchema,
		ColumnNames: []string{
			"equipment_id",
			"equipment_name",
			"equipment_acquisition_cost",
			"equipment_purchase_date",
		},
		RowsAsCsv: "",
		ExpectedResults: `{
  "errors": [
    {
      "code": "404",
      "description": "Resource with uniqueID '42' not found in equipment table"
    }
  ]
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("DELETE FROM (.+) WHERE (.+) = (.+)").
				WithArgs("42").
				WillReturnResult(sqlmock.NewResult(0, 0))
			mock.ExpectCommit()
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("DELETE", "/equipment/42", nil)
			return req
		}(),
	},
}
var successMsg = func(id, table string) []byte {
	text := fmt.Sprintf(
		"Resource with uniqueID '%s' successfully deleted from %s table",
		id,
		table,
	)
	msg := map[string]interface{}{
		"status": map[string]string{
			"code":        "200",
			"description": text,
		},
	}
	result, _ := json.MarshalIndent(&msg, "", "  ")
	return result
}
var failureMsg = func(id, table string) []byte {
	text := fmt.Sprintf(
		"Resource with uniqueID '%s' not found in %s table",
		id,
		table,
	)
	msg := map[string]interface{}{
		"errors": []map[string]string{
			map[string]string{
				"code":        "404",
				"description": text,
			},
		},
	}
	result, _ := json.MarshalIndent(&msg, "", "  ")
	return result
}

func (b *Backend) DeleteSingle(rw http.ResponseWriter, req *http.Request) {
	id := mux.Vars(req)["id"]
	routeName := mux.CurrentRoute(req).GetName()
	table := mux.Vars(req)["table"]
	requestTx := mux.Vars(req)["tx"]
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	queryTemplate := b.Templates[routeName]
	handler := &handler{
		vars: []string{queryTemplate},
		templateData: struct {
			TableName      string
			UniqueIDColumn string
		}{
			TableName:      table,
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

	// Check that user provided tx is already in backend.Transactions
	if requestTx != "" {
		tx, ok := b.Transactions.Load(requestTx)
		if !ok {
			http.Error(rw, string(failureMsg(requestTx, table)[:]), http.StatusNotFound)
			return
		}
		log.When(config.Options.Logging).Infof("Query will execute within user specified transaction:\n%s\n", tx)
	}
	result, err := b.execContext(req.Context(), queryString, id)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%#v\n",
		result,
	)
	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		rw.WriteHeader(http.StatusNotFound)
		rw.Write(failureMsg(id, table))
		return
	}
	rw.Write(successMsg(id, table))
	return
}
