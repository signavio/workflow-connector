package filter

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type Argument string
type Predicate string

const (
	Equal Predicate = "eq"
)

type Expression struct {
	Arguments []Argument
	Predicate Predicate
}

func New(ctx context.Context, filterQuery string) (*Expression, error) {
	queryString, err := url.QueryUnescape(filterQuery)
	if err != nil {
		return nil, fmt.Errorf("error unescaping query parameter: %s", err)
	}
	typeDescriptor := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		ctx.Value(util.ContextKey("table")).(string),
	)
	arguments, predicate, err := extractFrom(queryString, typeDescriptor)
	if err != nil {
		return nil, err
	}
	return &Expression{arguments, predicate}, nil
}
func extractFrom(filterQuery string, td *descriptor.TypeDescriptor) ([]Argument, Predicate, error) {
	parts := strings.Split(filterQuery, " ")
	columnName, err := getColumnNameIfExists(parts[0], td)
	if err != nil {
		return nil, "", err
	}
	switch Predicate(parts[1]) {
	case Equal:
		return []Argument{
				Argument(columnName),
				Argument(strings.Join(parts[2:], " ")),
			},
			Equal,
			nil
	default:
		return nil, "", fmt.Errorf(
			"operator or function '%s' is not supported",
			parts[1],
		)
	}
}
func getColumnNameIfExists(columnFromQuery string, td *descriptor.TypeDescriptor) (columnName string, err error) {
	for _, field := range td.Fields {
		if columnFromQuery == field.Key {
			columnName = field.FromColumn
			return columnName, nil
		}
	}
	return "", fmt.Errorf("column '%s' does not exist", columnName)
}
