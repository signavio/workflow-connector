package sql

import (
	"database/sql/driver"
	"net/http"
	"net/url"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var testCasesGetCollectionFilterable = []testCase{
	{
		Kind: "success",
		Name: "it succeeds when filtering equipment table using column name",
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
		RowsAsCsv: "3,Buntfink SteelKettle,39.95,2017-12-12T12:00:00Z",
		ExpectedResults: `{
  "acquisitionCost": {
    "amount": 39.95,
    "currency": "EUR"
  },
  "id": "3",
  "name": "Buntfink SteelKettle",
  "purchaseDate": "2017-12-12T12:00:00Z"
}`,
		ExpectedQueries: func(mock sqlmock.Sqlmock, columns []string, rowsAsCsv string, args ...driver.Value) {
			rows := sqlmock.NewRows(columns).
				FromCSVString(rowsAsCsv)
			mock.ExpectQuery("SELECT . FROM (.+) WHERE name = .").
				WillReturnRows(rows)
		},
		Request: func() *http.Request {
			req, _ := http.NewRequest("GET", "/equipment?filter=name+eq+Buntfink+SteelKettle", nil)
			return req
		},
	},
}

func TestExtractColumnNameFromFilterQueryParam(t *testing.T) {
	testCases := []struct {
		kind     string
		filter   string
		expected string
	}{
		{
			kind:     "success",
			filter:   url.QueryEscape("name eq Buntfink SteelKettle"),
			expected: "name",
		},
		{
			kind:   "failure",
			filter: url.QueryEscape("name+eq+Buntfink+SteelKettle"),
		},
		{
			kind:   "failure",
			filter: url.QueryEscape("foobar eq Buntfink SteelKettle"),
		},
	}
	for _, tc := range testCases {
		td := util.GetTypeDescriptorUsingTypeDescriptorKey(
			config.Options.Descriptor.TypeDescriptors,
			"equipment",
		)
		columnName, err := extractColumnNameFromFilterQueryParam(tc.filter, td)
		if tc.kind == "success" {
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			if columnName != tc.expected {
				t.Errorf("Expected columnName to be '%s' not '%s'", tc.expected, columnName)
				return
			}
		} else {
			if err == nil {
				t.Error("Expected error, got nil error")
				return
			}
		}
	}
}

func TestExtractOperatorFromFilterQueryParam(t *testing.T) {
	testCases := []struct {
		kind     string
		filter   string
		expected string
	}{
		{
			kind:     "success",
			filter:   url.QueryEscape("name eq Buntfink SteelKettle"),
			expected: "=",
		},
		{
			// Should fail if the user tries to use an operator
			// that is not yet implemented
			kind:   "failure",
			filter: url.QueryEscape("name lt Buntfink SteelKettle"),
		},
	}
	for _, tc := range testCases {
		op, err := extractOperatorFromFilterQueryParam(tc.filter)
		if tc.kind == "success" {
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			if op != tc.expected {
				t.Errorf("Expected operator  to be %v not '%s'", tc.expected, op)
				return
			}
		} else {
			if err == nil {
				t.Error("Expected error, got nil error")
				return
			}
		}
	}
}
