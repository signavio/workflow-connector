package sqltests

import (
	"database/sql/driver"
	"net/http"
	"net/url"
	"strings"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var (
	crudTests = map[string][]testCase{
		"GetSingle":     getSingleTestCases,
		"GetCollection": getCollectionTestCases,
		"CreateSingle":  createSingleTestCases,
		"UpdateSingle":  updateSingleTestCases,
		"DeleteSingle":  deleteSingleTestCases,
	}
	getSingleTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "2,Sanremo Café Racer,8477.85,2017-12-12T12:00:00Z",
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00.123Z",
  "recipes": [%s]
}`,
			ExpectedResultsRelationships: []interface{}{`
    {
      "creationDate": "2017-12-13T23:00:00.123Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "2017-01-13T00:00:00.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  `},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
					WithArgs("2").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/2", nil)
				return req
			},
		},
		{

			Kind: "failure",
			Name: "it fails and returns 404 NOT FOUND when querying a non existent equipment id",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "",
			ExpectedResults: `{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
%s`,
			ExpectedResultsRelationships: []interface{}{""},
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
			},
		},
		{
			Kind: "success",
			Name: "it succeeds when recipes table contains more than one column",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonRecipesTableSchema,
			ColumnNames: []string{
				"recipes\x00id",
				"recipes\x00equipment_id",
				"recipes\x00name",
				"recipes\x00instructions",
			},
			RowsAsCsv: "1,2,Espresso single shot,do this",
			ExpectedResults: `{
  "creationDate": "2017-12-13T23:00:00.123Z",
  "equipment": {%s},
  "equipmentId": 2,
  "id": "1",
  "instructions": "do this",
  "lastAccessed": "2017-01-13T00:00:00.000Z",
  "lastModified": "2017-12-14T00:00:00.123Z",
  "name": "Espresso single shot"
}`,
			ExpectedResultsRelationships: []interface{}{`
    "acquisitionCost": {
      "amount": 8477.85,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00.123Z"
  `},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
					WithArgs("1").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/recipes/1", nil)
				return req
			},
		},
	}
	getCollectionTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "1,Bialetti Moka Express 6 cup,25.95,2017-12-11T12:00:00Z\n" +
				"2,Sanremo Café Racer,8477.85,2017-12-12T12:00:00Z\n" +
				"3,Buntfink SteelKettle,39.95,2017-12-12T12:00:00Z\n" +
				"4,Copper Coffee Pot Cezve,49.95,2017-12-12T12:00:00Z",
			ExpectedResults: `[
  {
    "acquisitionCost": {
      "amount": 25.95,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Bialetti Moka Express 6 cup",
    "purchaseDate": "2017-12-11T12:00:00.123Z"
  },
  {
    "acquisitionCost": {
      "amount": 8477.85,
      "currency": "EUR"
    },
    "id": "2",
    "name": "Sanremo Café Racer",
    "purchaseDate": "2017-12-12T12:00:00.123Z"
  },
  {
    "acquisitionCost": {
      "amount": 39.95,
      "currency": "EUR"
    },
    "id": "3",
    "name": "Buntfink SteelKettle",
    "purchaseDate": "2017-12-12T12:00:00.000Z"
  },
  {
    "acquisitionCost": {
      "amount": 49.95,
      "currency": "EUR"
    },
    "id": "4",
    "name": "Copper Coffee Pot Cezve",
    "purchaseDate": "2017-12-12T12:00:00.000Z"
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
			},
		},
	}
	createSingleTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when provided with valid parameters as URL parameters",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "5,French Press,35.99,2017-04-02T00:00:00Z",
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 35.99,
    "currency": "EUR"
  },
  "id": "5",
  "name": "French Press",
  "purchaseDate": "2017-04-02T00:00:00.000Z",
  "recipes": [%s]
}`,
			ExpectedStatusCode: http.StatusCreated,
			ExpectedHeader: http.Header(map[string][]string{
				"Location": []string{"/equipment/5"},
			}),
			ExpectedResultsRelationships: []interface{}{""},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("INSERT INTO (.+)\\(id, name, acquisition_cost, purchase_date\\) VALUES\\(., ., ., .\\)").
					// insert id specifically instead of relying on the autoincrement feature
					// of the database. This allows us to run our tests multiple times on
					// the test database in such a way that the state of the database
					// before running the tests *is equal to* the state after
					// runnning the tests
					WithArgs("5", "French Press", "35.99", "2017-04-02T00:00:00Z").
					WillReturnResult(sqlmock.NewResult(5, 1))
				mock.ExpectCommit()
				rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
					WithArgs("5").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("id", "5")
				postData.Set("name", "French Press")
				postData.Set("acquisitionCost", "35.99")
				postData.Set("purchaseDate", "2017-04-02T00:00:00Z")
				req, _ := http.NewRequest("POST", "/equipment", strings.NewReader(postData.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
		},
	}
	updateSingleTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when provided with valid parameters as URL parameters",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "2,Sanremo Café Racer,9283.99,2017-12-12T12:00:00Z",
			// purchaseDate is rounded to the nearest second
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 9283.99,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-01T12:34:56.789Z",
  "recipes": [%s]
}`,
			ExpectedResultsRelationships: []interface{}{`
    {
      "creationDate": "2017-12-13T23:00:00.123Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "2017-01-13T00:00:00.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  `},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = ., purchase_date = . WHERE (.+) = .").
					WithArgs("Sanremo Café Racer", "9283.99", "2017-12-01T12:34:56.789Z", "2").
					WillReturnResult(sqlmock.NewResult(2, 1))
				mock.ExpectCommit()
				rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
					WithArgs("2").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("name", "Sanremo Café Racer")
				postData.Set("acquisitionCost", "9283.99")
				postData.Set("purchaseDate", "2017-12-01T12:34:56.789Z")
				req, _ := http.NewRequest("PATCH", "/equipment/2?"+postData.Encode(), nil)
				return req
			},
		},
		{
			Kind: "success",
			Name: "it succeeds when user explicitely wants to insert a null value",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "2,Sanremo Café Racer,8477.85,null",
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": null,
  "recipes": [%s]
}`,
			ExpectedResultsRelationships: []interface{}{`
    {
      "creationDate": "2017-12-13T23:00:00.123Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "2017-01-13T00:00:00.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  `},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = ., purchase_date = . WHERE (.+) = .").
					WithArgs("Sanremo Café Racer", 8477.85, "2017-12-12T12:00:00Z", "2").
					WillReturnResult(sqlmock.NewResult(2, 1))
				mock.ExpectCommit()
				rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
					WithArgs("2").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				body := strings.NewReader(
					`{"name": "Sanremo Café Racer", "acquisitionCost": 8477.85, "purchaseDate": null}`,
				)
				req, _ := http.NewRequest(
					"PATCH",
					"/equipment/2",
					body,
				)
				req.Header.Set("Content-Type", "application/json")
				return req
			},
		},
		{
			Kind: "success",
			Name: "it succeeds when provided with valid parameters as json in the request body",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "2,Sanremo Café Racer,8477.85,2017-12-12T12:00:00Z",
			ExpectedResults: `{
  "acquisitionCost": {
    "amount": 8477.85,
    "currency": "EUR"
  },
  "id": "2",
  "name": "Sanremo Café Racer",
  "purchaseDate": "2017-12-12T12:00:00.000Z",
  "recipes": [%s]
}`,
			ExpectedResultsRelationships: []interface{}{`
    {
      "creationDate": "2017-12-13T23:00:00.123Z",
      "equipmentId": 2,
      "id": "1",
      "instructions": "do this",
      "lastAccessed": "2017-01-13T00:00:00.000Z",
      "lastModified": "2017-12-14T00:00:00.123Z",
      "name": "Espresso single shot"
    }
  `},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = ., purchase_date = . WHERE (.+) = .").
					WithArgs("Sanremo Café Racer", 8477.85, "2017-12-12T12:00:00Z", "2").
					WillReturnResult(sqlmock.NewResult(2, 1))
				mock.ExpectCommit()
				rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
					WithArgs("2").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				body := strings.NewReader(
					`{"name": "Sanremo Café Racer", "acquisitionCost": 8477.85, "purchaseDate": "2017-12-12T12:00:00Z"}`,
				)
				req, _ := http.NewRequest(
					"PATCH",
					"/equipment/2",
					body,
				)
				req.Header.Set("Content-Type", "application/json")
				return req
			},
		},
		{

			Kind: "failure",
			Name: "it fails and returns 404 NOT FOUND when trying to update a non existent id",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "",
			ExpectedResults: `{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
%s`,
			ExpectedResultsRelationships: []interface{}{""},
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("UPDATE (.+) SET name = ., acquisition_cost = . WHERE (.+) = .").
					WithArgs("Sanremo Café Racer", "512.23", "42").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectCommit()
				rows := sqlmock.NewRows(columns).FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT . FROM (.+) AS (.+) WHERE (.+) = (.+)").
					WithArgs("42").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				postData := url.Values{}
				postData.Set("name", "Sanremo Café Racer")
				postData.Set("acquisitionCost", "512.23")
				req, _ := http.NewRequest("PATCH", "/equipment/42?"+postData.Encode(), nil)
				return req
			},
		},
	}
	deleteSingleTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds in deleting an existing resource",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "",
			ExpectedResults: `{
  "status": {
    "code": 200,
    "description": "Resource with uniqueID '5' successfully deleted from equipment table"
  }
}`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				mock.ExpectBegin()
				mock.ExpectExec("DELETE FROM (.+) WHERE (.+) = (.+)").
					WithArgs("5").
					WillReturnResult(sqlmock.NewResult(4, 1))
				mock.ExpectCommit()
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("DELETE", "/equipment/5", nil)
				return req
			},
		},
		{

			Kind: "failure",
			Name: "it fails and returns 404 NOT FOUND when trying to delete a non existent id",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			RowsAsCsv: "",
			ExpectedResults: `{
  "status": {
    "code": 404,
    "description": "Resource with uniqueID '42' not found in equipment table"
  }
}
`,
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
			},
		},
	}
)
