package formatting

import (
	"context"
	"encoding/json"

	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/log"
	"github.com/signavio/workflow-connector/pkg/util"
)

// JSONForWfa is the Formatter for Signavio Workflow Acccelerator
type JSONForWfa struct{}

// Format will convert the results received from the backend service,
// which is an array of empty interfaces, to a JSON byte array
// that Workflow Accelerator can interpret and understand
func (f *JSONForWfa) Format(ctx context.Context, cfg *config.Config, results []interface{}) (JSONResults []byte, err error) {
	activeRoute := ctx.Value(config.ContextKey("route")).(string)
	tableName := ctx.Value(config.ContextKey("table")).(string)
	if activeRoute == "getCollectionAsOptionsFilterable" ||
		activeRoute == "getSingleAsOption" ||
		activeRoute == "getCollectionAsOptions" {
		// Signavio Workflow Accelerator expects results from the options routes,
		// for example, `/options/{id}`, `/options?filter=`, to be enclosed
		// in an array, regardless of whether or note the result set
		// return 0, 1 or many results
		return specialFormattingForOptionsRoutes(results, tableName)
	}
	if len(results) == 0 {
		return []byte("{}"), nil
	}
	if len(results) == 1 {
		log.When(cfg).Infoln("[formatter -> asWorkflowType] Format with result set == 1")
		formattedResult := formatAsAWorkflowType(
			results[0].(map[string]interface{}), tableName, cfg,
		)
		log.When(cfg).Infof("[formatter <- asWorkflowType] formattedResult: \n%+v\n", formattedResult)
		JSONResults, err = json.MarshalIndent(&formattedResult, "", "  ")
		if err != nil {
			return nil, err
		}
		return
	}
	log.When(cfg).Infoln("[formatter -> asWorkflowType] Format with result set > 1")
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := formatAsAWorkflowType(
			result.(map[string]interface{}), tableName, cfg,
		)
		formattedResults = append(formattedResults, formattedResult)
	}
	log.When(cfg).Infof(
		"[formatter <- asWorkflowType] formattedResult (top 2): \n%+v ...\n",
		formattedResults[0:1],
	)
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	log.When(cfg).Infoln("[routeHandler <- formatter]")
	return
}

func formatAsAWorkflowType(nameValue map[string]interface{}, table string, cfg *config.Config) (result map[string]interface{}) {
	typeDescriptor := util.TypeDescriptorForCurrentTable(
		cfg.Descriptor.TypeDescriptors,
		table,
	)
	result = make(map[string]interface{})
	for _, field := range typeDescriptor.Fields {
		switch {
		case field.Type.Name == "money":
			if nameValue[table].(map[string]interface{})[field.Amount.FromColumn] != nil ||
				nameValue[table].(map[string]interface{})[field.Currency.FromColumn] != nil {
				result[field.Key] =
					resultAsWorkflowMoneyType(field, nameValue, table)
			}
		case field.Relationship != nil && nameValue[table].(map[string]interface{})[field.Key] != nil:
			if len(nameValue[table].(map[string]interface{})[field.Key].(map[string]interface{})[field.Relationship.WithTable].([]map[string]interface{})) > 0 {
				var results []map[string]interface{}
				relatedResults := nameValue[table].(map[string]interface{})[field.Key].(map[string]interface{})[field.Relationship.WithTable].([]map[string]interface{})
				for _, r := range relatedResults {
					results = append(results, formatAsAWorkflowType(
						map[string]interface{}{field.Relationship.WithTable: r},
						field.Relationship.WithTable,
						cfg,
					))
				}
				result[field.Key] = results
			} else {
				result[field.Key] = []interface{}{}
			}
		default:
			if nameValue[table].(map[string]interface{})[field.FromColumn] != nil {
				result[field.Key] =
					nameValue[table].(map[string]interface{})[field.FromColumn]
			}
		}
	}
	return
}

func resultAsWorkflowMoneyType(field *config.Field, nameValue map[string]interface{}, table string) map[string]interface{} {
	result := make(map[string]interface{})
	var currency interface{}
	if field.Currency.FromColumn == "" {
		if field.Currency.Value == "" {
			// Default to EUR if no other information is provided
			currency = "EUR"
		} else {
			// Otherwise use the currency that the user defines
			// in the `value` field
			currency = field.Currency.Value
		}
	} else {
		currency = nameValue[table].(map[string]interface{})[field.Currency.FromColumn]
	}
	result = map[string]interface{}{
		"amount":   nameValue[table].(map[string]interface{})[field.Amount.FromColumn],
		"currency": currency,
	}
	return result
}

func specialFormattingForOptionsRoutes(results []interface{}, table string) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("[{}]"), nil
	}
	var formattedResults []interface{}
	for _, result := range results {
		nameValue := result.(map[string]interface{})
		formattedResult := map[string]interface{}{
			"id": nameValue[table].(map[string]interface{})["id"],
			"name": nameValue[table].(map[string]interface{})["name"],
		}
		formattedResults = append(formattedResults, formattedResult)
	}
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	return

}

func specialFormattingForFilterRoute(results []interface{}, table string, cfg *config.Config) (JSONResults []byte, err error) {
	if len(results) == 0 {
		return []byte("[{}]"), nil
	}
	var formattedResults []interface{}
	for _, result := range results {
		formattedResult := formatAsAWorkflowType(
			result.(map[string]interface{}), table, cfg,
		)
		formattedResults = append(formattedResults, formattedResult)
	}
	JSONResults, err = json.MarshalIndent(&formattedResults, "", "  ")
	if err != nil {
		return nil, err
	}
	return
}
