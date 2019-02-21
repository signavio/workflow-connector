package sqltests

import (
	"encoding/json"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/config"
)

var (
	invalidEquipmentDescriptorFields = `
{
  "key" : "id",
  "name" : "ID",
  "fromColumn": "id",
  "type" : {
	"name" : "text"
  }
},

  "key" : "name",
  "name" : "Equipment Name",
  "fromColumn": "name",
  "type" : {
	"name" : "text"
  }
},
{
  "key" : "acquisitionCost",
  "name" : "Acquisition Cost",
  "type" : {
	"name" : "money",
	"amount" : {
      "key": "acquisitionCost",
	  "fromColumn": "acquisition_cost"
	},
	"currency" : {
	  "value" : "EUR"
	}
  }
},
{
  "key" : "purchaseDate",
  "name" : "Purchase Date",
  "fromColumn" : "purchase_date",
  "type" : {
	"name" : "date",
	"kind" : "date"
  }
},
{
  "key" : "recipes",
  "name" : "Associated recipes",
  "type" : {
  	"name": "text"
  },
  "relationship": {
  	"kind": "oneToMany",
  	"withTable": "recipes",
  	"localTableUniqueIdColumn": "id",
  	"foreignTableUniqueIdColumn": "equipment_id"
  }
}`
	conformityTests = map[string][]testCase{
		"GetDescriptor": getDescriptorTestCases,
	}
	getDescriptorTestCases = []testCase{
		{
			Kind: "success",
			Name: "it succeeds with proper descriptor file",
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
			ExpectedResults: []string{func() string {
				json, _ := json.MarshalIndent(config.Options.Descriptor, "", "  ")
				return string(json[:])
			}()},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				return req
			},
		},
		{

			Kind: "failure",
			Name: "it fails with invalid json",
			DescriptorFields: []string{
				invalidEquipmentDescriptorFields,
				commonRecipesDescriptorFields,
			},
			TableSchema: commonEquipmentTableSchema,
			ColumnNames: []string{
				"equipment\x00id",
				"equipment\x00name",
				"equipment\x00acquisition_cost",
				"equipment\x00purchase_date",
			},
			ExpectedStatusCodes: []int{http.StatusInternalServerError},
			ExpectedResults: []string{func() string {
				json, _ := json.MarshalIndent(config.Options.Descriptor, "", "  ")
				return string(json[:])
			}()},
			Request: func() *http.Request {
				req, _ := http.NewRequest("GET", "/", nil)
				return req
			},
		},
	}
)
