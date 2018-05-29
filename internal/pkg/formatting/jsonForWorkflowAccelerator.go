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

// Format will convert the results received from Theo backend service,
// which is an array of empty interfaces, to a JSON byte array
// that Workflow Accelerator can interpret and understand
func (f *workflowAcceleratorFormatter) Format(req *http.Request, results []interface{}) (JSONResults []byte, err error) {
	currentRoute := mux.CurrentRoute(req).GetName()
	tableName := mux.Vars(req)["table"]
	if currentRoute == "GetCollectionAsOptionsFilterable" ||
		currentRoute == "GetCollectionAsOptions" {
		// Signavio Workflow Accelerator expects results from the options routes,
		// for example, `/options`, `/options?filter=`, to be enclosed
		// in an array, regardless of whether or not the result set
		// return 0, 1 or many results
		return specialFormattingForCollectionAsOptionsRoutes(results, tableName)
	}
	if currentRoute == "GetSingleAsOption" {
		return specialFormattingForSingleAsOptionsRoute(results, tableName)
	}
	if len(results) == 0 {
		return []byte("{}"), nil
	}
	if len(results) == 1 {
		log.When(config.Options).Infoln("[formatter -> asWorkflowType] Format with result set == 1")
		formattedResult := formatAsAWorkflowType(
			results[0].(map[string]interface{}), tableName,
		)
		log.When(config.Options).Infof("[formatter <- asWorkflowType] formattedResult: \n%+v\n", formattedResult)
		JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
		if err != nil {
			return nil, err
		}
		return
	}
	log.When(config.Options).Infoln("[formatter -> asWorkflowType] Format with result set > 1")
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := formatAsAWorkflowType(
			result.(map[string]interface{}), tableName,
		)
		formattedResults = append(formattedResults, formattedResult)
	}
	log.When(config.Options).Infof(
		"[formatter <- asWorkflowType] formattedResult (top 2): \n%+v ...\n",
		formattedResults[0:1],
	)
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	log.When(config.Options).Infoln("[routeHandler <- formatter]")
	return
}

func formatAsAWorkflowType(queryResults map[string]interface{}, table string) (formatted map[string]interface{}) {
	typeDescriptor := util.GetTypeDescriptorUsingDBTableName(
		config.Options.Descriptor.TypeDescriptors,
		table,
	)
	formatted = make(map[string]interface{})
	for _, field := range typeDescriptor.Fields {
		formatted = buildResultFromQueryResultsUsingField(formatted, queryResults, table, field)
	}
	return
}

func buildResultFromQueryResultsUsingField(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	switch {
	case field.Type.Name == "money":
		formatted = buildForFieldTypeMoney(formatted, queryResults, table, field)
		return formatted
	case tableHasRelationships(queryResults, table, field):
		formatted = buildAndRecursivelyResolveRelationships(formatted, queryResults, table, field)
		return formatted
	default:
		formatted = buildForFieldTypeOther(formatted, queryResults, table, field)
		return formatted
	}
}

func tableHasRelationships(queryResults map[string]interface{}, table string, field *config.Field) bool {
	return field.Relationship != nil && queryResults[table].(map[string]interface{})[field.Key] != nil
}
func buildAndRecursivelyResolveRelationships(formatted, queryResults map[string]interface{}, table string, field *config.Field) map[string]interface{} {
	if hasRelatedTables(queryResults, table, field) {
		var results []map[string]interface{}
		relatedResults := queryResults[table].(map[string]interface{})[field.Key].(map[string]interface{})[field.Relationship.WithTable].([]map[string]interface{})
		for _, r := range relatedResults {
			results = append(results, formatAsAWorkflowType(
				map[string]interface{}{field.Relationship.WithTable: r},
				field.Relationship.WithTable,
			))
		}
		formatted[field.Key] = results
	} else {
		formatted[field.Key] = []interface{}{}
	}
	return formatted
}

func hasRelatedTables(queryResults map[string]interface{}, table string, field *config.Field) bool {
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
func specialFormattingForSingleAsOptionsRoute(results []interface{}, table string) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("{}"), nil
	}
	formattedResult := mapWithIDAndName(results[0].(map[string]interface{}), table)
	JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
	if err != nil {
		return nil, err
	}
	return
}

func specialFormattingForCollectionAsOptionsRoutes(results []interface{}, table string) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("[{}]"), nil
	}
	if len(results) == 1 {
		formattedResult := mapWithIDAndName(results[0].(map[string]interface{}), table)
		JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
		if err != nil {
			return nil, err
		}
		return
	}
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := mapWithIDAndName(result.(map[string]interface{}), table)
		formattedResults = append(formattedResults, formattedResult)
	}
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	return

}

func mapWithIDAndName(queryResults map[string]interface{}, table string) map[string]interface{} {
	id := queryResults[table].(map[string]interface{})["id"]
	var name interface{}
	switch v := queryResults[table].(map[string]interface{})["name"].(type) {
	case int64:
		name = fmt.Sprintf("%v", v)
	case float64:
		name = fmt.Sprintf("%v", v)
	case time.Time:
		name = v.String()
	case string:
		name = v
	}
	return map[string]interface{}{
		"id":   id,
		"name": name,
	}
}
