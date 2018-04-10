package sql

import (
	"bytes"
	"context"
	"html/template"

	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/util"
)

// TODO: from here
func (b *Backend) interpolateGetTemplate(ctx context.Context, templateText, tableName string) (interpolatedQuery string, err error) {
	queryTemplate, err := template.New("dbquery").Parse(templateText)
	if err != nil {
		return "", err
	}
	query := bytes.NewBufferString("")
	templateData := struct {
		TableName string
		Relations []*config.Field
	}{
		TableName: tableName,
		Relations: ctx.Value(config.ContextKey("relationships")).([]*config.Field),
	}
	err = queryTemplate.Execute(query, templateData)
	if err != nil {
		return "", err
	}
	return query.String(), nil
}

func (b *Backend) interpolateTemplate(ctx context.Context, templateText string, requestData map[string]interface{}) (interpolatedQuery string, err error) {
	var columnNamesFromRequestData []string
	currentTable := ctx.Value(config.ContextKey("table")).(string)
	td := util.TypeDescriptorForCurrentTable(b.Cfg.Descriptor.TypeDescriptors, currentTable)
	for _, field := range td.Fields {
		if field.Type.Name == "money" {
			if _, ok := requestData[field.Amount.Key]; ok {
				columnNamesFromRequestData = append(columnNamesFromRequestData, field.Amount.FromColumn)
			}
			if _, ok := requestData[field.Currency.Key]; ok {
				columnNamesFromRequestData = append(columnNamesFromRequestData, field.Currency.FromColumn)
			}
		} else {
			if _, ok := requestData[field.Key]; ok {
				columnNamesFromRequestData = append(columnNamesFromRequestData, field.FromColumn)
			}
		}
	}
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
	}
	queryTemplate, err := template.New("dbquery").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", err
	}
	query := bytes.NewBufferString("")
	templateData := struct {
		Table       string
		ColumnNames []string
	}{
		Table:       currentTable,
		ColumnNames: columnNamesFromRequestData,
	}
	if len(templateData.ColumnNames) == 0 {
		return "", ErrPostForm
	}
	err = queryTemplate.Execute(query, templateData)
	if err != nil {
		return "", err
	}
	return query.String(), nil
}
