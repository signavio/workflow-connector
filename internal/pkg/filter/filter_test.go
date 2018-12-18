package filter

import (
	"context"
	"net/url"
	"testing"

	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type testCase struct {
	kind     string
	filter   string
	expected *Expression
}

var testCases = []testCase{
	{
		kind:   "success",
		filter: url.QueryEscape("name eq Buntfink SteelKettle"),
		expected: &Expression{
			Arguments: []Argument{"name", "Buntfink SteelKettle"},
			Predicate: Predicate("eq"),
		},
	},
	{
		kind:   "failure",
		filter: url.QueryEscape("name+eq+Buntfink+SteelKettle"),
	},
	{
		kind:   "failure",
		filter: url.QueryEscape("foobar+eq+Buntfink+SteelKettle"),
	},
	// using an unsupported operator
	{
		kind:   "failure",
		filter: url.QueryEscape("name lt Buntfink SteelKettle"),
	},
}

func TestFilter(t *testing.T) {
	for _, tc := range testCases {
		ctx := context.WithValue(
			context.Background(),
			util.ContextKey("table"),
			"equipment",
		)
		expression, err := New(ctx, tc.filter)
		if tc.kind == "success" {
			if err != nil {
				t.Errorf("Expected no error, got %v", err)
				return
			}
			if expression.Arguments[0] != tc.expected.Arguments[0] {
				t.Errorf("Expected columnName to be '%s' not '%s'", tc.expected.Arguments[0], expression.Arguments[0])
				return
			}
			if expression.Arguments[1] != tc.expected.Arguments[1] {
				t.Errorf("Expected Value to be '%s' not '%s'", tc.expected.Arguments[1], expression.Arguments[1])
				return
			}
			if expression.Predicate != tc.expected.Predicate {
				t.Errorf("Expected Predicate to be '%s' not '%s'", tc.expected.Predicate, expression.Predicate)
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
