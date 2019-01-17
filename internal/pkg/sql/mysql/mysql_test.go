package mysql

import (
	"fmt"
	"testing"

	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
)

// TestCase for sql backend
type testCase struct {
	// A testCase should assert success cases or failure cases
	Kind string
	// A testCase has a unique name
	Name string
	// A testCase has an array of fields
	Fields []*descriptor.Field
	// A testCase has a SQL query that will be modified to include the database specific date coersion function
	QueryToCoerce string
	// A testCase contains an array of columnNames
	ColumnNames    []string
	ExpectedResult string
}

var (
	commonColumnNames = []string{"id", "name", "purchase_date", "acquisition_cost"}
	commonFields      = []*descriptor.Field{
		&descriptor.Field{
			Key:  "id",
			Name: "Equipment Id",
			Type: &descriptor.WorkflowType{
				Name: "text",
			},
			FromColumn: "id",
		},
		&descriptor.Field{
			Key:  "name",
			Name: "Equipment Name",
			Type: &descriptor.WorkflowType{
				Name: "text",
			},
			FromColumn: "name",
		},
		&descriptor.Field{
			Key:  "purchaseDate",
			Name: "Purchase Date",
			Type: &descriptor.WorkflowType{
				Name: "date",
				Kind: "date",
			},
			FromColumn: "purchase_date",
		},
		&descriptor.Field{
			Key:  "acquisitionCost",
			Name: "Acquisition Cost",
			Type: &descriptor.WorkflowType{
				Name: "money",
				Amount: &descriptor.Amount{
					Key:        "acquisitionCost",
					FromColumn: "acquisition_cost",
				},
				Currency: &descriptor.Currency{
					Value: "EUR",
				},
			},
		},
	}
	testCases = []*testCase{
		&testCase{
			Kind:        "success",
			Name:        "it only wraps columns of `datetime` type with database specific date coersion function on update",
			Fields:      commonFields,
			ColumnNames: commonColumnNames,
			QueryToCoerce: `UPDATE "equipment" ` +
				`SET "name" = ?, ` +
				`SET "acquisition_cost" = ?, ` +
				`SET "purchase_date" = ?, ` +
				`WHERE "id" = ?;`,
			ExpectedResult: `UPDATE "equipment" ` +
				`SET "name" = ?, ` +
				`SET "acquisition_cost" = ?, ` +
				`SET "purchase_date" =  str_to_date(?, '%Y-%m-%dT%TZ'), ` +
				`WHERE "id" = ?;`,
		},
		&testCase{
			Kind:        "success",
			Name:        "it only wraps columns of `datetime` type with database specific date coersion function on create",
			ColumnNames: []string{"id", "name", "acquisition_cost", "purchase_date"},
			Fields:      commonFields,
			QueryToCoerce: `INSERT INTO equipment(id, name, acquisition_cost, purchase_date) ` +
				`VALUES (?, ?, ?, ?)`,
			ExpectedResult: `INSERT INTO equipment(id, name, acquisition_cost, purchase_date) ` +
				`VALUES (?, ?, ?,  str_to_date(?, '%Y-%m-%dT%TZ'))`,
		},
		&testCase{
			Kind:        "success",
			Name:        "it successfully handles column names containing a literal question mark character `?`",
			ColumnNames: []string{"id?", "name?", "pur?chase_date", "?acquisition_cost'"},
			Fields: []*descriptor.Field{
				&descriptor.Field{
					Key:  "id",
					Name: "Equipment Id",
					Type: &descriptor.WorkflowType{
						Name: "text",
					},
					FromColumn: "id?",
				},
				&descriptor.Field{
					Key:  "name",
					Name: "Equipment Name",
					Type: &descriptor.WorkflowType{
						Name: "text",
					},
					FromColumn: "name?",
				},
				&descriptor.Field{
					Key:  "purchaseDate",
					Name: "Purchase Date",
					Type: &descriptor.WorkflowType{
						Name: "date",
						Kind: "date",
					},
					FromColumn: "pur?chase_date",
				},
				&descriptor.Field{
					Key:  "acquisitionCost",
					Name: "Acquisition Cost",
					Type: &descriptor.WorkflowType{
						Name: "money",
						Amount: &descriptor.Amount{
							Key:        "acquisitionCost",
							FromColumn: "?acquisition_cost'",
						},
						Currency: &descriptor.Currency{
							Value: "EUR",
						},
					},
				},
			},
			QueryToCoerce: `UPDATE "equipment" ` +
				`SET "name?" = ?, ` +
				`SET "?acquisition_cost'" = ?, ` +
				`SET "pur?chase_date" = ?, ` +
				`WHERE "id" = ?;`,
			ExpectedResult: `UPDATE "equipment" ` +
				`SET "name?" = ?, ` +
				`SET "?acquisition_cost'" = ?, ` +
				`SET "pur?chase_date" =  str_to_date(?, '%Y-%m-%dT%TZ'), ` +
				`WHERE "id" = ?;`,
		},
	}
)

func TestCoerceExecArgsToMysqlType(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			if err := run(tc); err != nil {
				t.Error(err)
			}
		})
	}
}

func run(tc *testCase) error {
	switch tc.Kind {
	case "success":
		if err := itSucceeds(tc); err != nil {
			return err
		}
		return nil
	case "failure":
		if err := itFails(tc); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("testcase should either be success or failure kind")
	}
}
func itFails(tc *testCase) error {
	return nil
}

func itSucceeds(tc *testCase) error {
	gotResult := coerceExecArgsToMysqlType(tc.QueryToCoerce, tc.ColumnNames, tc.Fields)
	if gotResult != tc.ExpectedResult {
		return fmt.Errorf("expected:\n%s\ngot:\n%s\n", tc.ExpectedResult, gotResult)
	}
	return nil
}
