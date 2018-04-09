package sql

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/util"
)

type templatesTestCases struct {
	name           string
	b              *Backend
	expectedResult string
	ctx            context.Context
	descriptor     string
}

var testCasesForDescriptorWithZeroRelationship = func() []templatesTestCases {
	descriptor := `{
  "key": "maintenanceConnector",
  "name": "Maintenace Scheduling Connector",
  "description": "Signavio Workflow Accelerator integration with our internal databases to help us automate maintenance checks for our equipment",
  "typeDescriptors": [
    {
      "key" : "equipment",
      "name" : "Equipment",
      "tableName": "equipment",
      "columnAsOptionName": "name",
      "fields" : [
        {
          "key" : "name",
          "name" : "Name",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "cost",
          "name" : "Acquisition Cost",
          "type" : {
            "name" : "money",
            "amount": {
                "fromColumn": "acquisition_cost"
            },
            "currency": {
                "value": "EUR"
            }
          }
        },
        {
          "key" : "purchase_date",
          "name" : "Purchase Date",
          "type" : {
            "name" : "date",
            "kind" : "date"
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ],
  "version": 1,
  "protocolVersion": 1
}`
	return []templatesTestCases{
		{
			name:           "GetSingleWithRelationships",
			b:              setupInterpolationTests(descriptor),
			expectedResult: "SELECT * FROM equipment AS _equipment WHERE _equipment.id = ?",
		},
	}
}

var testCasesForDescriptorWithOneRelationship = func() []templatesTestCases {
	descriptor := `{
  "key": "maintenanceConnector",
  "name": "Maintenace Scheduling Connector",
  "description": "Signavio Workflow Accelerator integration with our internal databases to help us automate maintenance checks for our equipment",
  "typeDescriptors": [
    {
      "key" : "equipment",
      "name" : "Equipment",
      "tableName": "equipment",
      "columnAsOptionName": "name",
      "fields" : [
        {
          "key" : "name",
          "name" : "Name",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "cost",
          "name" : "Acquisition Cost",
          "type" : {
            "name" : "money",
            "amount": {
                "fromColumn": "acquisition_cost"
            },
            "currency": {
                "value": "EUR"
            }
          }
        },
        {
          "key" : "purchase_date",
          "name" : "Purchase Date",
          "type" : {
            "name" : "date",
            "kind" : "date"
          }
        },
        {
          "key" : "equipment_maintenance",
          "name" : "Maintenance performed",
          "type" : {
            "name" : "text"
          },
          "relationship": {
            "withTable": "maintenance",
            "foreignKey": "equipment_id",
            "denormalizeData": true
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    },
    {
      "key" : "maintenance",
      "name" : "Maintenance",
      "tableName": "maintenance",
      "columnAsOptionName": "id",
      "fields" : [
        {
          "key" : "equipment_id",
          "name" : "Equipment ID",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "maintenance_performed",
          "name" : "Maintenance Performed",
          "type" : {
            "name" : "date",
            "kind" : "datetime"
          }
        },
        {
          "key" : "notes",
          "name" : "Notes",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "isScheduled",
          "name" : "Is scheduled?",
          "type" : {
            "name" : "boolean"
          }
        },
        {
          "key" : "next_maintenance",
          "name" : "Next Maintenance Date",
          "type" : {
            "name" : "date",
            "kind" : "datetime"
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ],
  "version": 1,
  "protocolVersion": 1
}`
	return []templatesTestCases{
		{
			name: "GetSingleWithRelationships",
			b:    setupInterpolationTests(descriptor),
			expectedResult: "SELECT * FROM equipment AS _equipment LEFT JOIN maintenance" +
				" ON maintenance.equipment_id = _equipment.id" +
				" WHERE _equipment.id = ?",
		},
	}
}

var testCasesForDescriptorWithTwoRelationships = func() []templatesTestCases {
	descriptor := `{
  "key": "maintenanceConnector",
  "name": "Maintenace Scheduling Connector",
  "description": "Signavio Workflow Accelerator integration with our internal databases to help us automate maintenance checks for our equipment",
  "typeDescriptors": [
    {
      "key" : "equipment",
      "name" : "Equipment",
      "tableName": "equipment",
      "columnAsOptionName": "name",
      "fields" : [
        {
          "key" : "name",
          "name" : "Name",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "cost",
          "name" : "Acquisition Cost",
          "type" : {
            "name" : "money",
            "amount": {
                "fromColumn": "acquisition_cost"
            },
            "currency": {
                "value": "EUR"
            }
          }
        },
        {
          "key" : "purchase_date",
          "name" : "Purchase Date",
          "type" : {
            "name" : "date",
            "kind" : "date"
          }
        },
        {
          "key" : "equipment_maintenance",
          "name" : "Maintenance performed",
          "type" : {
            "name" : "text"
          },
          "relationship": {
            "withTable": "maintenance",
            "foreignKey": "equipment_id",
            "denormalizeData": true
          }
        },
        {
          "key" : "equipment_warranty",
          "name" : "Warranty on Equipment",
          "type" : {
            "name" : "text"
          },
          "relationship": {
            "withTable": "warranty",
            "foreignKey": "equipment_id",
            "denormalizeData": true
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    },
    {
      "key" : "maintenance",
      "name" : "Maintenance",
      "tableName": "maintenance",
      "columnAsOptionName": "id",
      "fields" : [
        {
          "key" : "equipment_id",
          "name" : "Equipment ID",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "maintenance_performed",
          "name" : "Maintenance Performed",
          "type" : {
            "name" : "date",
            "kind" : "datetime"
          }
        },
        {
          "key" : "notes",
          "name" : "Notes",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "isScheduled",
          "name" : "Is scheduled?",
          "type" : {
            "name" : "boolean"
          }
        },
        {
          "key" : "next_maintenance",
          "name" : "Next Maintenance Date",
          "type" : {
            "name" : "date",
            "kind" : "datetime"
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    },
    {
      "key" : "warranty",
      "name" : "Warrany",
      "tableName": "warranty",
      "columnAsOptionName": "id",
      "fields" : [
        {
          "key" : "equipment_id",
          "name" : "Equipment ID",
          "type" : {
            "name" : "text"
          }
        },
        {
          "key" : "start_date",
          "name" : "Warranty start date",
          "type" : {
            "name" : "date",
            "kind" : "datetime"
          }
        },
        {
          "key" : "duration_in_years",
          "name" : "Warranty duration [year(s)]",
          "type" : {
            "name" : "number"
          }
        },
        {
          "key" : "conditions",
          "name" : "Warranty conditions",
          "type" : {
            "name" : "text"
          }
        }
      ],
      "optionsAvailable" : true,
      "fetchOneAvailable" : true
    }
  ],
  "version": 1,
  "protocolVersion": 1
}`
	return []templatesTestCases{
		{
			name: "GetSingleWithRelationships",
			b:    setupInterpolationTests(descriptor),
			expectedResult: "SELECT * FROM equipment AS _equipment LEFT JOIN maintenance" +
				" ON maintenance.equipment_id = _equipment.id" +
				" LEFT JOIN warranty ON warranty.equipment_id = _equipment.id" +
				" WHERE _equipment.id = ?",
		},
	}
}

func setupInterpolationTests(descriptor string) *Backend {
	cfg := config.Initialize(
		strings.NewReader(descriptor))
	backend := NewBackend(cfg, mux.NewRouter())
	backend.Templates["GetSingleWithRelationships"] =
		"SELECT * FROM {{.TableName}} AS _{{.TableName}}" +
			"{{range .Relations}}" +
			" LEFT JOIN {{.Relationship.WithTable}}" +
			" ON {{.Relationship.WithTable}}.{{.Relationship.ForeignKey}}" +
			" = _{{$.TableName}}.id{{end}}" +
			" WHERE _{{$.TableName}}.id = ?"
	return backend
}

func TestInterpolateTemplates(t *testing.T) {
	testCases := []struct {
		name string
		tcs  []templatesTestCases
	}{
		{
			name: "descriptor with zero relationships",
			tcs:  testCasesForDescriptorWithZeroRelationship(),
		},
		{
			name: "descriptor with one relationship",
			tcs:  testCasesForDescriptorWithOneRelationship(),
		},
		{
			name: "descriptor with two relationships",
			tcs:  testCasesForDescriptorWithTwoRelationships(),
		},
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("when using %s", tc.name), func(t *testing.T) {
			for _, ttc := range tc.tcs {
				t.Run(fmt.Sprintf("using template %v", ttc.name), func(t *testing.T) {
					contextWithRelationships := util.ContextWithRelationships(
						context.Background(),
						ttc.b.Cfg.Descriptor.TypeDescriptors,
						"equipment",
					)
					ttc.ctx = contextWithRelationships
					assertExpectations(ttc, t)
				})
			}
		})
	}
}

func assertExpectations(tc templatesTestCases, t *testing.T) {
	interpolatedQuery, err := tc.b.interpolateGetTemplate(
		tc.ctx,
		tc.b.Templates[tc.name],
		"equipment",
	)
	if err != nil {
		t.Errorf("Unexpected error occured: %v", err)
	}
	if interpolatedQuery != tc.expectedResult {
		t.Errorf(
			"Interpolated Query:\n%v\nShould equal expected result:\n%v\n",
			interpolatedQuery,
			tc.expectedResult,
		)
	}
}
