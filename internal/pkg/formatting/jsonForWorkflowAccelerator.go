package formatting

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/descriptor"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

type standardFormatter struct{}
type getSingleAsOptionFormatter struct{}
type getCollectionAsOptionsFormatter struct{}

var (
	Standard               = &standardFormatter{}
	GetSingleAsOption      = &getSingleAsOptionFormatter{}
	GetCollectionAsOptions = &getCollectionAsOptionsFormatter{}
)

// Format will convert the results received from the backend service,
// which is an array of empty interfaces, to a JSON byte array
// that Workflow Accelerator can interpret and understand
func (f *standardFormatter) Format(req *http.Request, results []interface{}) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("{}"), nil
	}
	tableName := req.Context().Value(util.ContextKey("table")).(string)
	if len(results) == 1 {
		log.When(config.Options.Logging).Infoln("[formatter -> asWorkflowType] Format with result set == 1")
		formattedResult := formatAsAWorkflowType(
			results[0].(map[string]interface{}), req, tableName,
		)
		log.When(config.Options.Logging).Infof("[formatter <- asWorkflowType] formattedResult: \n%+v\n", formattedResult)
		JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
		if err != nil {
			return nil, err
		}
		log.When(config.Options.Logging).Infoln("[routeHandler <- formatter]")
		return
	}
	log.When(config.Options.Logging).Infoln("[formatter -> asWorkflowType] Format with result set > 1")
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := formatAsAWorkflowType(
			result.(map[string]interface{}), req, tableName,
		)
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
func (f *getSingleAsOptionFormatter) Format(req *http.Request, results []interface{}) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("{}"), nil
	}
	if len(results) > 1 {
		return nil, fmt.Errorf("formatting: expected result set to contain only one resource")
	}
	tableName := req.Context().Value(util.ContextKey("table")).(string)
	formattedResult := make(map[string]interface{})
	formattedResult["id"] = results[0].(map[string]interface{})[tableName].(map[string]interface{})["id"]
	formattedResult["name"] = results[0].(map[string]interface{})[tableName].(map[string]interface{})["name"]
	log.When(config.Options.Logging).Infof("[formatter <- asWorkflowType] formattedResult: \n%+v\n", formattedResult)
	JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
	if err != nil {
		return nil, err
	}
	log.When(config.Options.Logging).Infoln("[routeHandler <- formatter]")
	return
}

// formatForOptionsRoute applies special formatting to the results from
// the options routes, for example, `/options`, `/options?filter=`,
// to be enclosed in an array, regardless of whether or not
// the result set return 0, 1 or many results
func (f *getCollectionAsOptionsFormatter) Format(req *http.Request, results []interface{}) (JSONResults []byte, err error) {
	tableName := req.Context().Value(util.ContextKey("table")).(string)
	var formattedResults []interface{}
	if len(results) == 0 {
		return []byte("[]"), nil
	}
	for _, result := range results {
		formattedResult := make(map[string]interface{})
		formattedResult["id"] = result.(map[string]interface{})[tableName].(map[string]interface{})["id"]
		formattedResult["name"] = result.(map[string]interface{})[tableName].(map[string]interface{})["name"]
		formattedResults = append(formattedResults, formattedResult)
	}
	log.When(config.Options.Logging).Infof(
		"[formatter <- asWorkflowType] formattedResult(s): \n%+v ...\n",
		formattedResults,
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

func buildResultFromQueryResultsWithoutRelationships(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *descriptor.Field) map[string]interface{} {
	if field.Type.Name == "money" {
		formatted = buildForFieldTypeMoney(formatted, queryResults, table, field)
		return formatted
	}
	if field.Type.Kind == "date" {
		formatted = buildForFieldTypeDate(formatted, queryResults, table, field)
		return formatted
	}
	if field.Type.Kind == "datetime" {
		formatted = buildForFieldTypeDateTime(formatted, queryResults, table, field)
		return formatted
	}
	if field.FromColumn == req.Context().Value(util.ContextKey("uniqueIDColumn")).(string) {
		formatted = buildForFieldTypeUniqueIdColumn(formatted, queryResults, table, field)
		return formatted
	}
	formatted = buildForFieldTypeOther(formatted, queryResults, table, field)
	return formatted
}
func buildResultFromQueryResultsUsingField(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *descriptor.Field) map[string]interface{} {
	if tableHasRelationships(queryResults, table, field) {
		formatted = buildAndRecursivelyResolveRelationships(formatted, queryResults, req, table, field)
		return formatted
	}
	return buildResultFromQueryResultsWithoutRelationships(formatted, queryResults, req, table, field)
}

func tableHasRelationships(queryResults map[string]interface{}, table string, field *descriptor.Field) bool {
	return field.Relationship != nil && queryResults[table].(map[string]interface{})[field.Key] != nil
}
func buildAndRecursivelyResolveRelationships(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *descriptor.Field) map[string]interface{} {
	switch field.Relationship.Kind {
	case "oneToMany":
		return relationshipKindIsOneToMany(formatted, queryResults, req, table, field)
	case "manyToOne", "oneToOne":
		return relationshipKindIsXToOne(formatted, queryResults, req, table, field)
	default:
		return make(map[string]interface{})
	}
}

func relationshipKindIsOneToMany(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *descriptor.Field) map[string]interface{} {
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

func relationshipKindIsXToOne(formatted, queryResults map[string]interface{}, req *http.Request, table string, field *descriptor.Field) map[string]interface{} {
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
func relatedTablesResultSetNotEmpty(queryResults map[string]interface{}, table string, field *descriptor.Field) bool {
	fieldKey := queryResults[table].(map[string]interface{})[field.Key].(map[string]interface{})
	fieldKeyRelationshipWithTable := fieldKey[field.Relationship.WithTable].([]map[string]interface{})
	return len(fieldKeyRelationshipWithTable) > 0
}
func buildForFieldTypeMoney(formatted, queryResults map[string]interface{}, table string, field *descriptor.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.Type.Amount.FromColumn] != nil ||
		queryResults[table].(map[string]interface{})[field.Type.Currency.FromColumn] != nil {
		formatted[field.Key] =
			resultAsWorkflowMoneyType(field, queryResults, table)
	}
	return formatted
}
func buildForFieldTypeDate(formatted, queryResults map[string]interface{}, table string, field *descriptor.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		dateTime := queryResults[table].(map[string]interface{})[field.FromColumn].(time.Time)
		// Don't convert dateTime to UTC since when a DATE type is coerced
		// into a *time.Time it can contain the database's timezone.
		// Converting the dateTime to UTC can change the original
		// date from 2006-01-02T00:00:00+01:00 to
		// 2006-01-01T23:00:00+0:00 when in UTC
		formatted[field.Key] = dateTime.Format("2006-01-02T15:04:05.999Z")
	}
	return formatted
}
func buildForFieldTypeDateTime(formatted, queryResults map[string]interface{}, table string, field *descriptor.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		dateTime := queryResults[table].(map[string]interface{})[field.FromColumn].(time.Time)
		formatted[field.Key] = dateTime.UTC().Format("2006-01-02T15:04:05.999Z")
	}
	return formatted
}
func buildForFieldTypeUniqueIdColumn(formatted, queryResults map[string]interface{}, table string, field *descriptor.Field) map[string]interface{} {
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
func buildForFieldTypeOther(formatted, queryResults map[string]interface{}, table string, field *descriptor.Field) map[string]interface{} {
	if queryResults[table].(map[string]interface{})[field.FromColumn] != nil {
		formatted[field.Key] =
			queryResults[table].(map[string]interface{})[field.FromColumn]
	}
	return formatted
}
func resultAsWorkflowMoneyType(field *descriptor.Field, queryResults map[string]interface{}, table string) map[string]interface{} {
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
