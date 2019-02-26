package query

import (
	"bytes"
	"context"
	"text/template"
	"time"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type QueryTemplate struct {
	Vars               []string
	TemplateData       interface{}
	ColumnNames        []string
	CoerceExecArgsFunc func(string, []string, []*descriptor.Field) string
}

func (e *QueryTemplate) Interpolate(ctx context.Context, requestData map[string]interface{}) (interpolatedQuery string, args []interface{}, err error) {
	templateText := e.Vars[0]
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
		"format": func(a string) string {
			return a
		},
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
	args, err = CoerceRequestDataToGolangNativeTypes(ctx, requestData)
	if err != nil {
		return "", nil, err
	}
	return query.String(), args, nil

}
func CoerceRequestDataToGolangNativeTypes(ctx context.Context, requestData map[string]interface{}) (args []interface{}, err error) {
	currentTable := ctx.Value(util.ContextKey("table")).(string)
	td := util.GetTypeDescriptorUsingDBTableName(config.Options.Descriptor.TypeDescriptors, currentTable)
	var val interface{}
	var ok bool
	appendRequestDataToArgs := func(args []interface{}, val interface{}) []interface{} {
		switch v := val.(type) {
		case string:
			return append(args, v)
		case bool:
			return append(args, v)
		case float64:
			return append(args, v)
		case time.Time:
			return append(args, v)
		case nil:
			return append(args, nil)
		}
		return []interface{}{}
	}
	for _, field := range td.Fields {
		switch field.Type.Name {
		case "money":
			if val, ok = requestData[field.Type.Amount.Key]; ok {
				if val == nil {
					args = appendRequestDataToArgs(args, nil)
				} else {
					args = appendRequestDataToArgs(args, val)
				}
			}
			if val, ok = requestData[field.Type.Currency.Key]; ok {
				args = appendRequestDataToArgs(args, val)
			}
		case "datetime", "date", "time":
			if val, ok = requestData[field.Key]; ok {
				if val == nil {
					args = appendRequestDataToArgs(args, nil)
				} else {
					stringifiedDateTime := val.(string)
					parsedDateTime, err := time.ParseInLocation(
						"2006-01-02T15:04:05.999Z", stringifiedDateTime, time.UTC,
					)
					if err != nil {
						return nil, err
					}
					args = appendRequestDataToArgs(args, parsedDateTime)
				}
			}
		default:
			if val, ok = requestData[field.Key]; ok {
				if val == nil {
					args = appendRequestDataToArgs(args, nil)
				} else {
					args = appendRequestDataToArgs(args, val)
				}
			}
		}
	}
	return
}
