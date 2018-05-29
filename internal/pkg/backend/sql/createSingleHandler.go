package sql

import (
	"context"
	"database/sql/driver"
	"fmt"
	"net/http"
	"net/url"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var TestCasesCreateSingle = []TestCase{
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
		RowsAsCsv: "4,Cooling Spiral,99.99,2017-03-02T00:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 99.99,
    "currency": "EUR"
  },
  "id": "4",
  "name": "Cooling Spiral",
  "purchaseDate": "2017-03-02T00:00:00Z"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			mock.ExpectBegin()
			mock.ExpectExec("INSERT INTO (.+)\\(id, name, acquisition_cost\\) VALUES\\(., ., .\\)").
				WithArgs("Cooling Spiral", "99.99").
				WillReturnResult(sqlmock.NewResult(4, 1))
			mock.ExpectCommit()
			rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
				WithArgs("4").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			postData := url.Values{}
			postData.Set("id", "4")
			postData.Set("name", "Cooling Spiral")
			postData.Set("acquisitionCost", "99.99")
			postData.Set("purchaseDate", "2017-03-02T00:00:00Z")
			req, _ := http.NewRequest("POST", "/equipment?"+postData.Encode(), nil)
			return req
		}(),
	},
}

func (b *Backend) CreateSingle(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := mux.Vars(req)["table"]
	requestTx := mux.Vars(req)["tx"]
	queryTemplate := b.Templates[routeName]
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
			TableName   string
			ColumnNames []string
		}{
			TableName:   table,
			ColumnNames: columnNames,
		},
	}
	log.When(config.Options).Infof("[handler] %s\n", routeName)

	log.When(config.Options).Infoln("[handler -> template] interpolate query string")
	queryString, args, err := handler.interpolateExecTemplates(req.Context(), requestData)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options).Infof("[handler <- template]\n%s\n", queryString)
	log.When(config.Options).Infof("will be called with these args:\n%s\n", args)

	// Check that user provided tx is already in backend.Transactions
	if requestTx != "" {
		tx, ok := b.Transactions.Load(requestTx)
		if !ok {
			http.Error(rw, ErrTransactionUUIDInvalid.Error(), http.StatusInternalServerError)
			return
		}
		log.When(config.Options).Infof("Query will execute within user specified transaction:\n%s\n", tx)
	}
	result, err := b.execContext(req.Context(), queryString, args...)
	if err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options).Infof("[handler <- db] query results: \n%#v\n", result)

	log.When(config.Options).Infoln("[handler] try to return the newly updated resource")
	lastInsertID, err := result.LastInsertId()
	if err != nil || lastInsertID < 1 {
		// LastInsertId() probably not supported by the database. Therefore,
		// Since we can not return the newly created resource to the user,
		// we instead return an empty body and a 204 No Content
		log.When(config.Options).Infof(
			"[handler] Returning newly updated resource not supported by %s database\n",
			config.Options.Database.Driver,
		)
		rw.WriteHeader(http.StatusNoContent)
		return
	}
	updatedRoute := context.WithValue(
		req.Context(),
		util.ContextKey("currentRoute"),
		"GetSingle",
	)
	isCreated := context.WithValue(
		updatedRoute,
		util.ContextKey("isCreated"),
		true,
	)
	usingLastInsertID := context.WithValue(
		isCreated,
		util.ContextKey("id"),
		fmt.Sprintf("%d", lastInsertID),
	)
	newReq := req.WithContext(usingLastInsertID)
	b.GetSingle(rw, newReq)
	return
	//queryString, err = interpolateCreateSingle(req.Context(), "GetSingleWithRelationships", table)
	//if err != nil {
	//	http.Error(rw, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//rows, err := b.DB.QueryContext(
	//	req.Context(),
	//	queryString,
	//	fmt.Sprintf("%d", lastInsertId),
	//)
	//if err != nil {
	//	http.Error(rw, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//defer rows.Close()
	//results, err := rowsToResults(rows, columnNames, dataTypes)
	//if err != nil {
	//	http.Error(rw, err.Error(), http.StatusInternalServerError)
	//	return
	//}
	//log.When(config.Options).Infof(
	//	"[handler <- db] getQueryResults: \n%+v\n",
	//	results,
	//)
	//log.When(config.Options).Infoln("[routeHandler <- handlers]")
	//log.When(config.Options).Infoln("[routeHandler -> formatter]")
}

func getColumnNamesFromRequestData(tableName string, requestData map[string]interface{}) (columnNames []string) {
	td := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		tableName,
	)
	for _, field := range td.Fields {
		if field.Type.Name == "money" {
			if _, ok := requestData[field.Type.Amount.Key]; ok {
				columnNames = append(columnNames, field.Type.Amount.FromColumn)
			}
			if _, ok := requestData[field.Type.Currency.Key]; ok {
				columnNames = append(columnNames, field.Type.Currency.FromColumn)
			}
		} else {
			if _, ok := requestData[field.Key]; ok {
				columnNames = append(columnNames, field.FromColumn)
			}
		}
	}
	return
}
