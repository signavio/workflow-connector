package sqltests

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var (
	dataConnectorOptionsTests = map[string][]testCase{
		"GetSingleAsOption":                getSingleAsOptionTestCases,
		"GetCollectionAsOptions":           getCollectionAsOptionsTestCases,
		"GetCollectionAsOptionsFilterable": getCollectionAsOptionsFilterableTestCases,
		"GetCollectionAsOptionsWithParams": getCollectionAsOptionsWithParamsTestCases,
	}
	getSingleAsOptionTestCases = []testCase{
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
			},
			RowsAsCsv: "1,Bialetti Moka Express 6 cup",
			ExpectedResults: `{
  "id": "1",
  "name": "Bialetti Moka Express 6 cup"
}`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT (.+), (.+) FROM  (.+) WHERE (.+) = (.+)").
					WithArgs("1").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options/1", nil)
				return req
			},
		},
		{
			Kind: "failure",
			Name: "it fails and returns 404 NOT FOUND when querying a non existent id",
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
	}
	getCollectionAsOptionsTestCases = []testCase{
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
			},
			RowsAsCsv: "1,Bialetti Moka Express 6 cup\n" +
				"2,Sanremo Café Racer\n" +
				"3,Buntfink SteelKettle\n" +
				"4,Copper Coffee Pot Cezve",
			ExpectedResults: `[
  {
    "id": "1",
    "name": "Bialetti Moka Express 6 cup"
  },
  {
    "id": "2",
    "name": "Sanremo Café Racer"
  },
  {
    "id": "3",
    "name": "Buntfink SteelKettle"
  },
  {
    "id": "4",
    "name": "Copper Coffee Pot Cezve"
  }
]`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT (.+), (.+) FROM (.+)").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options", nil)
				return req
			},
		},
	}
	getCollectionAsOptionsWithParamsTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column" +
				" and returns three records when we filter on purchaseDate",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
			},
			RowsAsCsv: "2,Sanremo Café Racer\n3,Buntfink SteelKettle\n4, Copper Coffee Pot Cezve",
			ExpectedResults: `[
  {
    "id": "2",
    "name": "Sanremo Café Racer"
  },
  {
    "id": "3",
    "name": "Buntfink SteelKettle"
  },
  {
    "id": "4",
    "name": "Copper Coffee Pot Cezve"
  }
]`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT (.+), (.+) FROM (.+) WHERE (.+) LIKE (.+)").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=&purchaseDate=2017-12-12T12:00:00Z", nil)
				return req
			},
		},
		{
			Kind: "success",
			Name: "it succeeds when equipment table contains more than one column" +
				" and returns one record when we filter on purchaseDate and provide" +
				" a filter query",
			DescriptorFields: []string{
				commonEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
			},
			RowsAsCsv: "2,Sanremo Café Racer",
			ExpectedResults: `[
  {
    "id": "2",
    "name": "Sanremo Café Racer"
  }
]`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT (.+), (.+) FROM (.+) WHERE (.+) LIKE (.+)").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=San&purchaseDate=2017-12-12T12:00:00Z", nil)
				return req
			},
		},
	}
	getCollectionAsOptionsFilterableTestCases = []testCase{
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
			},
			RowsAsCsv: "1,Bialetti Moka Express 6 cup",
			ExpectedResults: `[
  {
    "id": "1",
    "name": "Bialetti Moka Express 6 cup"
  }
]`,
			ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
				rows := sqlmock.NewRows(columns).
					FromCSVString(rowsAsCsv)
				mock.ExpectQuery("SELECT (.+), (.+) FROM (.+) WHERE (.+) LIKE (.+)").
					WillReturnRows(rows)
			},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/equipment/options?filter=moka", nil)
				return req
			},
		},
	}
)
