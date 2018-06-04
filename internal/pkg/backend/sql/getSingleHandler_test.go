package sql

import (
	"database/sql/driver"
	"net/http"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

var testCasesGetSingle = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when equipment table contains more than one column",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonMaintenanceDescriptorFields,
		},
		TableSchema: commonEquipmentTableSchema,
		ColumnNames: []string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		RowsAsCsv: "1,Stainless Steel Mash Tun (50L),999,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 999,
    "currency": "EUR"
  },
  "id": "1",
  "maintenance": [%s],
  "name": "Stainless Steel Mash Tun (50L)",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedResultsRelationships: []interface{}{`
    {
      "comments": "It went well!",
      "datePerformed": "2018-02-03T12:22:01Z",
      "dateScheduled": "2017-02-03T02:00:00Z",
      "equipmentId": 1,
      "id": "1",
      "maintainerId": 1
    },
    {
      "comments": "It went okay!",
      "datePerformed": "2018-02-03T12:22:01Z",
      "dateScheduled": "2017-02-03T02:00:00Z",
      "equipmentId": 1,
      "id": "3",
      "maintainerId": 2
    }
  `},
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
		},
	},
	{

		Kind: "failure",
		Name: "it fails and returns 404 NOT FOUND when querying a non existent equipment id",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonMaintenanceDescriptorFields,
		},
		TableSchema: commonEquipmentTableSchema,
		ColumnNames: []string{
			"equipment\x00id",
			"equipment\x00name",
			"equipment\x00acquisition_cost",
			"equipment\x00purchase_date",
		},
		RowsAsCsv:                    "",
		ExpectedResults:              `%s`,
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
		Name: "it succeeds when maintenance table contains more than one column",
		DescriptorFields: []string{
			commonEquipmentDescriptorFields,
			commonMaintenanceDescriptorFields,
		},
		TableSchema: commonMaintenanceTableSchema,
		ColumnNames: []string{
			"maintenance\x00id",
			"maintenance\x00date_scheduled",
			"maintenance\x00date_performed",
			"maintenance\x00equipment_id",
			"maintenance\x00maintainer_id",
			"maintenance\x00comments",
		},
		RowsAsCsv: "1,2017-02-03T02:00:00Z,2018-02-03T12:22:01Z,1,1,It went well!",
		ExpectedResults: `{
  "comments": "It went well!",
  "datePerformed": "2018-02-03T12:22:01Z",
  "dateScheduled": "2017-02-03T02:00:00Z",
  "equipmentId": {%s},
  "id": "1",
  "maintainerId": {%s}
}`,
		ExpectedResultsRelationships: []interface{}{`
    "acquisitionCost": {
      "amount": 999,
      "currency": "EUR"
    },
    "id": "1",
    "name": "Stainless Steel Mash Tun (50L)",
    "purchaseDate": "2017-12-12T12:00:00Z"
  `, `
    "emailAddress": "jane.feather@example.com",
    "familyName": "Feather",
    "id": "1",
    "preferredName": "Jane"
  `,
		},
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE (.+) = (.+)").
				WithArgs("1").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/maintenance/1", nil)
			return req
		},
	},
}
