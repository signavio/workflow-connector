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
	commonColumnNames = []string{"name", "purchase_date", "acquisition_cost"}
	commonFields      = []*descriptor.Field{
		&descriptor.Field{
			Key:  "name",
			Name: "Equipment Name",
			Type: &descriptor.WorkflowType{
				Name: "text",
				Amount: &descriptor.Amount{
					FromColumn: "",
				},
			},
			FromColumn: "name",
		},
		&descriptor.Field{
			Key:  "purchaseDate",
			Name: "Purchase Date",
			Type: &descriptor.WorkflowType{
				Name: "date",
				Kind: "date",
				Amount: &descriptor.Amount{
					FromColumn: "",
				},
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
			Name:        "it only wraps columns of `datetime` type with database specific date coersion function",
			Fields:      commonFields,
			ColumnNames: commonColumnNames,
			QueryToCoerce: `UPDATE "equipment" ` +
				`SET "name" = ?, ` +
				`SET "purchase_date" = ?, ` +
				`SET "acquisition_cost" = ?, ` +
				`WHERE "id" = ?;`,
			ExpectedResult: `UPDATE "equipment" ` +
				`SET "name" = ?, ` +
				`SET "purchase_date" = str_to_date(?, '%Y-%m-%dT%TZ'), ` +
				`SET "acquisition_cost" = ?, ` +
				`WHERE "id" = ?;`,
		},
		&testCase{
			Kind:        "success",
			Name:        "it successfully handles column names containing a literal question mark character `?`",
			ColumnNames: []string{"name?", "pur?chase_date", "?acquisition_cost'"},
			Fields: []*descriptor.Field{
				&descriptor.Field{
					Key:  "name",
					Name: "Equipment Name",
					Type: &descriptor.WorkflowType{
						Name: "text",
						Amount: &descriptor.Amount{
							FromColumn: "",
						},
					},
					FromColumn: "name?",
				},
				&descriptor.Field{
					Key:  "purchaseDate",
					Name: "Purchase Date",
					Type: &descriptor.WorkflowType{
						Name: "date",
						Kind: "date",
						Amount: &descriptor.Amount{
							FromColumn: "",
						},
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
				`SET "pur?chase_date" = ?, ` +
				`SET "?acquisition_cost'" = ?, ` +
				`WHERE "id" = ?;`,
			ExpectedResult: `UPDATE "equipment" ` +
				`SET "name?" = ?, ` +
				`SET "pur?chase_date" = str_to_date(?, '%Y-%m-%dT%TZ'), ` +
				`SET "?acquisition_cost'" = ?, ` +
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
