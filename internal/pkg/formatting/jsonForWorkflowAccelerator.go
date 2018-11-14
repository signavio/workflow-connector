package formatting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type workflowAcceleratorFormatter struct{}

// WorkflowAcccelerator will format the results retrieved from
// the database to comply with Workflow Accelerator's API
var WorkflowAccelerator = &workflowAcceleratorFormatter{}

// Format will convert the results received from the backend service,
// which is an array of empty interfaces, to a JSON byte array
// that Workflow Accelerator can interpret and understand
func (f *workflowAcceleratorFormatter) Format(req *http.Request, results []interface{}) (JSONResults []byte, err error) {
	tableName := req.Context().Value(util.ContextKey("table")).(string)
	uniqueIDColumn := req.Context().Value(util.ContextKey("uniqueIDColumn")).(string)
	columnAsOptionName := req.Context().Value(util.ContextKey("columnAsOptionName")).(string)
	if len(results) == 0 {
		// Signavio Workflow Accelerator expects results from the options routes,
		// for example, `/options`, `/options?filter=`, to be enclosed
		// in an array, regardless of whether or not the result set
		// return 0, 1 or many results
		if isOptionsRoute(req) {
			return []byte("[{}]"), nil
		}
		return []byte("{}"), nil
	}
	if len(results) == 1 {
		log.When(config.Options.Logging).Infoln("[formatter -> asWorkflowType] Format with result set == 1")
		formattedResult := formatAsAWorkflowType(
			results[0].(map[string]interface{}), req, tableName,
		)
		log.When(config.Options.Logging).Infof("[formatter <- asWorkflowType] formattedResult: \n%+v\n", formattedResult)
		if isOptionsRoute(req) {
			var optionResults []interface{}
			optionResult := map[string]interface{}{
				"id":   formattedResult[uniqueIDColumn],
				"name": formattedResult[columnAsOptionName],
			}
			optionResults = append(optionResults, optionResult)
			JSONResults, err = json.MarshalIndent(&optionResults, "", "  ")
			if err != nil {
				return nil, err
			}
			return
		}
		if isOptionRoute(req) {
			optionResult := map[string]interface{}{
				"id":   formattedResult[uniqueIDColumn],
				"name": formattedResult[columnAsOptionName],
			}
			JSONResults, err = json.MarshalIndent(&optionResult, "", "  ")
			if err != nil {
				return nil, err
			}
			return
		}
		JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
		if err != nil {
			return nil, err
		}
		return
	}
	log.When(config.Options.Logging).Infoln("[formatter -> asWorkflowType] Format with result set > 1")
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := formatAsAWorkflowType(
			result.(map[string]interface{}), req, tableName,
		)
		if isOptionRoute(req) || isOptionsRoute(req) {
			formattedResult = map[string]interface{}{
				"id":   formattedResult[uniqueIDColumn],
				"name": formattedResult[columnAsOptionName],
			}
		}
		formattedResults = append(formattedResults, formattedResult)
	}
	log.When(config.Options.Logging).Infof(
		"[formatter <- asWorkflowType] formattedResult (top 2): \n%+v ...\n",
		formattedResults[0:1],
	)
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	log.When(config.Options.Logging).Infoln("[routeHandler <- formatter]")
	return
}

func formatAsAWorkflowType(queryResults map[string]interface{}, req *http.Request, table string) (formatted map[string]interface{}) {
	typeDescriptor := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		table,
	)
	formatted = make(map[string]interface{})
	for _, field := range typeDescriptor.Fields {
		if mux.CurrentRoute(req).GetName() == "GetCollection" {
			formatted = buildResultFromQueryResultsWithoutRelationships(
				formatted, queryResults, req, table, field,
			)
		} else {
			formatted = buildResultFromQueryResultsUsingField(
				formatted, queryResults, req, table, field,
			)
		}
	}
	return
}

func buildResultFromQueryResultsWithoutRelationships(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *config.Field) map[string]interface{} {
	if field.Type.Name == "money" {
		formatted = buildForFieldTypeMoney(formatted, queryResults, table, field)
		return formatted
	}
	if field.Type.Name == "date" {
		formatted = buildForFieldTypeDate(formatted, queryResults, table, field)
		return formatted
	}
	if field.FromColumn == req.Context().Value(util.ContextKey("uniqueIDColumn")).(string) {
		formatted = buildForFieldTypeUniqueIdColumn(formatted, queryResults, table, field)
		return formatted
	}
	formatted = buildForFieldTypeOther(formatted, queryResults, table, field)
	return formatted
}
func buildResultFromQueryResultsUsingField(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *config.Field) map[string]interface{} {
	if tableHasRelationships(queryResults, table, field) {
		formatted = buildAndRecursivelyResolveRelationships(formatted, queryResults, req, table, field)
		return formatted
	}
	return buildResultFromQueryResultsWithoutRelationships(formatted, queryResults, req, table, field)
}

func tableHasRelationships(queryResults map[string]interface{}, table string, field *config.Field) bool {
	return field.Relationship != nil && queryResults[table].(map[string]interface{})[field.Key] != nil
}
func buildAndRecursivelyResolveRelationships(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *config.Field) map[string]interface{} {
	switch field.Relationship.Kind {
	case "oneToMany":
		return relationshipKindIsOneToMany(formatted, queryResults, req, table, field)
	case "manyToOne", "oneToOne":
		return relationshipKindIsXToOne(formatted, queryResults, req, table, field)
	default:
		return make(map[string]interface{})
	}
}

func relationshipKindIsOneToMany(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *config.Field) map[string]interface{} {
	if relatedTablesResultSetNotEmpty(queryResults, table, field) {
		var results []map[string]interface{}
		relatedResults := queryResults[table].(map[string]interface{})[field.Key].(map[string]interface{})[field.Relationship.WithTable].([]map[string]interface{})
		for _, r := range relatedResults {
			results = append(results, formatAsAWorkflowType(
				map[string]interface{}{field.Relationship.WithTable: r},
				req,
				field.Relationship.WithTable,
			))
		}
		formatted[field.Key] = results
	} else {
		formatted[field.Key] = []interface{}{}
	}
	return formatted
}

func relationshipKindIsXToOne(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *config.Field) map[string]interface{} {
	if relatedTablesResultSetNotEmpty(queryResults, table, field) {
		var result map[string]interface{}
		relatedResults := queryResults[table].(map[string]interface{})[field.Key].(map[string]interface{})[field.Relationship.WithTable].([]map[string]interface{})
		result = formatAsAWorkflowType(
			map[string]interface{}{field.Relationship.WithTable: relatedResults[0]},
			req,
			field.Relationship.WithTable,
		)
		formatted[field.Key] = result
		return formatted
	}
	formatted[field.Key] = make(map[string]interface{})
	return formatted
}
func relatedTablesResultSetNotEmpty(queryResults map[string]interface{}, table string, field *config.Field) bool {
	fieldKey := queryResults[table].(map[string]interface{})[field.Key].(map[string]interface{})
	fieldKeyRelationshipWithTable := fieldKey[field.Relationship.WithTable].([]map[string]interface{})
	return len(fieldKeyRelationshipWithTable) > 0
}
func buildForFieldTypeMoney(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.Type.Amount.FromColumn] != nil ||
		queryResults[table].(map[string]interface{})[field.Type.Currency.FromColumn] != nil {
		formatted[field.Key] =
			resultAsWorkflowMoneyType(field, queryResults, table)
	}
	return formatted
}
func buildForFieldTypeDate(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		dateTime := queryResults[table].(map[string]interface{})[field.FromColumn].(time.Time)
		formatted[field.Key] = dateTime.UTC().Format("2006-01-02T15:04:05.999Z")
	}
	return formatted
}
func buildForFieldTypeUniqueIdColumn(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		var uniqueIDColumn interface{}
		switch v := queryResults[table].(map[string]interface{})[field.FromColumn].(type) {
		case int64:
			uniqueIDColumn = fmt.Sprintf("%v", v)
		case float64:
			uniqueIDColumn = fmt.Sprintf("%v", v)
		case time.Time:
			uniqueIDColumn = v.String()
		case string:
			uniqueIDColumn = v
		}
		formatted[field.Key] = uniqueIDColumn
		return formatted
	}
	return formatted
}
func buildForFieldTypeOther(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		formatted[field.Key] =
			queryResults[table].(map[string]interface{})[field.FromColumn]
	}
	return formatted
}
func resultAsWorkflowMoneyType(field *config.Field, queryResults map[string]interface{}, table string) map[string]interface{} {
	result := make(map[string]interface{})
	var currency interface{}
	if field.Type.Currency.FromColumn == "" {
		if field.Type.Currency.Value == "" {
			// Default to EUR if no other information is provided
			currency = "EUR"
		} else {
			// Otherwise use the currency that the user defines
			// in the `value` field
			currency = field.Type.Currency.Value
		}
	} else {
		currency = queryResults[table].(map[string]interface{})[field.Type.Currency.FromColumn]
	}
	result = map[string]interface{}{
		"amount":   queryResults[table].(map[string]interface{})[field.Type.Amount.FromColumn],
		"currency": currency,
	}
	return result
}
func isOptionsRoute(req *http.Request) bool {
	currentRoute := mux.CurrentRoute(req).GetName()
	if currentRoute == "GetCollectionAsOptionsFilterable" ||
		currentRoute == "GetCollectionAsOptions" {
		return true
	}
	return false
}

func isOptionRoute(req *http.Request) bool {
	currentRoute := mux.CurrentRoute(req).GetName()
	if currentRoute == "GetSingleAsOption" {
		return true
	}
	return false
}
