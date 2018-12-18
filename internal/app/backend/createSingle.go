package backend

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/query"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) CreateSingle(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	table := req.Context().Value(util.ContextKey("table")).(string)
	queryTemplateUninterpolated := b.GetQueryTemplate(routeName)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	requestData, err := util.ParseDataForm(req)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}
	columnNames := getColumnNamesFromRequestData(table, requestData)
	if len(columnNames) == 0 {
		msg := &util.ResponseMessage{
			Code: http.StatusBadRequest,
			Msg: fmt.Sprintf(
				"the request data contains *one or more* fields "+
					"that are not present in the database\n"+
					"request data:\n%v\n"+
					"fields available in database table:\n%v\n",
				requestData, b.GetSchemaMapping(table).FieldNames),
		}
		http.Error(rw, msg.Error(), http.StatusBadRequest)
		return
	}
	execTemplate := &query.ExecTemplate{
		QueryTemplate: query.QueryTemplate{
			Vars: []string{queryTemplateUninterpolated},
			TemplateData: struct {
				TableName      string
				ColumnNames    []string
				UniqueIDColumn string
			}{
				TableName:      table,
				ColumnNames:    columnNames,
				UniqueIDColumn: uniqueIDColumn,
			},
		},
		ColumnNames:        columnNames,
		CoerceExecArgsFunc: b.GetCoerceExecArgsFunc(),
	}
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	log.When(config.Options.Logging).Infoln("[handler -> backend] interpolate query string")
	queryString, args, err := execTemplate.Interpolate(req.Context(), requestData)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	result, err := b.ExecContext(req.Context(), queryString, args...)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	log.When(config.Options.Logging).Infof("[handler <- db] query results: \n%s\n", result)

	log.When(config.Options.Logging).Infoln("[handler] try to return the newly updated resource")
	lastInsertID, err := result.LastInsertId()
	if err != nil || lastInsertID < 1 {
		// LastInsertId() probably not supported by the database. Therefore,
		// Since we can not return the newly created resource to the user,
		// we instead return an empty body and a 204 No Content
		log.When(config.Options.Logging).Infof(
			"[handler] returning newly updated resource not supported by %s database\n",
			config.Options.Database.Driver,
		)
		msg := &util.ResponseMessage{
			Code: http.StatusNoContent,
			Msg:  "resource succesfully created",
		}
		rw.WriteHeader(http.StatusNoContent)
		rw.Write(msg.Byte())
		return
	}
	updatedRoute := context.WithValue(
		req.Context(),
		util.ContextKey("currentRoute"),
		"GetSingle",
	)
	isCreated := context.WithValue(
		updatedRoute,
		util.ContextKey("isCreated"),
		true,
	)
	usingLastInsertID := context.WithValue(
		isCreated,
		util.ContextKey("id"),
		fmt.Sprintf("%d", lastInsertID),
	)
	newReq := req.WithContext(usingLastInsertID)
	b.GetSingle(rw, newReq)
	return
}

func getColumnNamesFromRequestData(tableName string, requestData map[string]interface{}) (columnNames []string) {
	td := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		tableName,
	)
	for _, field := range td.Fields {
		if field.Type.Name == "money" {
			if _, ok := requestData[field.Type.Amount.Key]; ok {
				columnNames = append(columnNames, field.Type.Amount.FromColumn)
			}
			if _, ok := requestData[field.Type.Currency.Key]; ok {
				columnNames = append(columnNames, field.Type.Currency.FromColumn)
			}
		} else {
			if _, ok := requestData[field.Key]; ok {
				columnNames = append(columnNames, field.FromColumn)
			}
		}
	}
	return
}
