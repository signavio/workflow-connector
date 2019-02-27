package query

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type QueryTemplate struct {
	Vars             []string
	TemplateData     interface{}
	ColumnNames      []string
	CoerceArgFuncs   map[string]func(map[string]interface{}, *descriptor.Field) (interface{}, bool, error)
	QueryFormatFuncs map[string]func() string
}

func (e *QueryTemplate) Interpolate(ctx context.Context, requestData map[string]interface{}) (interpolatedQuery string, args []interface{}, err error) {
	templateText := e.Vars[0]
	currentTable := ctx.Value(util.ContextKey("table")).(string)

	funcMap := template.FuncMap{
		"add2": func(x int) int {
			return x + 2
		},
		"lenPlus1": func(x []string) int {
			return len(x) + 1
		},
		"head": func(x []string) string {
			return x[0]
		},
		"tail": func(x []string) []string {
			return x[1:]
		},
		"format": func(tableName string, queryFormatFuncs map[string]func() string) func(int, string) string {
			return func(idx int, columnName string) string {

				nextIdx := fmt.Sprintf(":%d", idx+2)
				td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, tableName)
				for _, field := range td.Fields {
					switch field.Type.Name {
					case "money":
						if field.Type.Amount.FromColumn == columnName {
							return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
						}
						if field.Type.Currency.FromColumn == columnName {
							return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
						}
					case "date":
						if field.FromColumn == columnName {
							if field.Type.Kind == "datetime" {
								return fmt.Sprintf(queryFormatFuncs["datetime"](), nextIdx)
							}
							if field.Type.Kind == "date" {
								return fmt.Sprintf(queryFormatFuncs["date"](), nextIdx)
							}
							if field.Type.Kind == "time" {
								return fmt.Sprintf(queryFormatFuncs["time"](), nextIdx)
							}
						}
					default:
						if field.FromColumn == columnName {
							return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
						}
					}
				}
				return nextIdx
			}
		}(currentTable, e.QueryFormatFuncs),
	}
	queryTemplate, err := template.New("dbquery").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", nil, err
	}
	query := bytes.NewBufferString("")
	err = queryTemplate.Execute(query, e.TemplateData)
	if err != nil {
		return "", nil, err
	}
	args, err = CoerceRequestDataToGolangNativeTypes(ctx, requestData, e.CoerceArgFuncs)
	if err != nil {
		return "", nil, err
	}
	return query.String(), args, nil

}
func CoerceRequestDataToGolangNativeTypes(ctx context.Context, requestData map[string]interface{}, coerceArgFuncs map[string]func(map[string]interface{}, *descriptor.Field) (interface{}, bool, error)) (args []interface{}, err error) {
	currentTable := ctx.Value(util.ContextKey("table")).(string)
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, currentTable)
	for _, field := range td.Fields {
		switch field.Type.Name {
		case "money":
			result, ok, err := coerceArgFuncs["money"](requestData, field)
			if err != nil {
				return args, err
			}
			if ok {
				args = append(args, result)
			}
		case "datetime":
			result, ok, err := coerceArgFuncs["datetime"](requestData, field)
			if err != nil {
				return args, err
			}
			if ok {
				args = append(args, result)
			}
		case "date":
			switch field.Type.Kind {
			case "date":
				result, ok, err := coerceArgFuncs["date"](requestData, field)
				if err != nil {
					return args, err
				}
				if ok {
					args = append(args, result)
				}
			case "datetime":
				result, ok, err := coerceArgFuncs["datetime"](requestData, field)
				if err != nil {
					return args, err
				}
				if ok {
					args = append(args, result)
				}
			case "time":
				result, ok, err := coerceArgFuncs["time"](requestData, field)
				if err != nil {
					return args, err
				}
				if ok {
					args = append(args, result)
				}
			}
		default:
			result, ok, err := coerceArgFuncs["default"](requestData, field)
			if err != nil {
				return args, err
			}
			if ok {
				args = append(args, result)
			}
		}
	}
	return
}

func format(idx int, columnName string, tableName string, queryFormatFuncs map[string]func() string) string {
	nextIdx := fmt.Sprintf(":%d", idx+2)
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, tableName)
	for _, field := range td.Fields {
		switch field.Type.Name {
		case "money":
			if field.Type.Amount.FromColumn == columnName {
				return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
			}
			if field.Type.Currency.Key == columnName {
				return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
			}
		case "date":
			if field.FromColumn == columnName {
				if field.Type.Kind == "datetime" {
					return fmt.Sprintf(queryFormatFuncs["datetime"](), nextIdx)
				}
				if field.Type.Kind == "date" {
					return fmt.Sprintf(queryFormatFuncs["date"](), nextIdx)
				}
				if field.Type.Kind == "time" {
					return fmt.Sprintf(queryFormatFuncs["time"](), nextIdx)
				}
			}
		default:
			if field.FromColumn == columnName {
				return fmt.Sprintf(queryFormatFuncs["default"](), nextIdx)
			}
		}
	}
	return nextIdx
}
