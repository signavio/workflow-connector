package sql

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	"github.com/sdaros/workflow-db-connector/pkg/config"
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

func (b *Backend) interpolateTemplate(ctx context.Context, templateText string) (interpolatedQuery string, err error) {
	var columnNamesFromRequestData []string
	currentTable := ctx.Value(config.ContextKey("table")).(string)
	for _, column := range b.Cfg.TableSchemas[currentTable].ColumnNames {
		// Remove tablename prefix
		tableNamePrefix := strings.IndexRune(column, '_')
		columnName := column[tableNamePrefix+1 : len(column)]
		if _, ok := b.RequestData[columnName]; ok {
			columnNamesFromRequestData = append(columnNamesFromRequestData, columnName)
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
		Table:       ctx.Value(config.ContextKey("table")).(string),
		ColumnNames: columnNamesFromRequestData,
	}
	if len(templateData.ColumnNames) == 0 {
		return "", ErrPostFormEmpty
	}
	err = queryTemplate.Execute(query, templateData)
	if err != nil {
		return "", err
	}
	return query.String(), nil
}
