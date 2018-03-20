package util

import (
	"context"
	"net/http"
	"reflect"

	"github.com/signavio/workflow-connector/pkg/config"
)

// TypeDescriptorForCurrentTable will return the typeDescriptor from the descriptor.json
// file defined for the table provided in the function's second parameter
func TypeDescriptorForCurrentTable(typeDescriptors []*config.TypeDescriptor, table string) (typeDescriptor *config.TypeDescriptor) {
	var result *config.TypeDescriptor
	for _, typeDescriptor := range typeDescriptors {
		if table == typeDescriptor.TableName {
			result = typeDescriptor
		}
	}
	return result
}

// BuildRequest will add necessary key value pairs to the context in request
// that will be used later
func BuildRequest(req *http.Request, typeDescriptors []*config.TypeDescriptor, table string) (*http.Request, context.CancelFunc) {
	contextWithCurrentTable := context.WithValue(
		req.Context(),
		config.ContextKey("table"),
		table)

	contextWithColumnAsOptionName := context.WithValue(
		contextWithCurrentTable,
		config.ContextKey("columnAsOptionName"),
		TypeDescriptorForCurrentTable(typeDescriptors, table).ColumnAsOptionName)

	contextWithRelationships := ContextWithRelationships(
		contextWithColumnAsOptionName,
		typeDescriptors,
		table)

	contextWithCancel, cancelFn := context.WithCancel(contextWithRelationships)

	return req.WithContext(contextWithCancel), cancelFn
}

// BuildRequest will add necessary key value pairs to the context in request
// that will be used later
func BuildContext(ctx context.Context, typeDescriptors []*config.TypeDescriptor, table string) context.Context {
	contextWithCurrentTable := context.WithValue(
		ctx,
		config.ContextKey("table"),
		table)

	contextWithColumnAsOptionName := context.WithValue(
		contextWithCurrentTable,
		config.ContextKey("columnAsOptionName"),
		TypeDescriptorForCurrentTable(typeDescriptors, table).ColumnAsOptionName)

	contextWithRelationships := ContextWithRelationships(
		contextWithColumnAsOptionName,
		typeDescriptors,
		table)

	return contextWithRelationships
}

// ContextWithRelationships will return a new context which will included an array of all relationships for the table provided in the function's second parameter
func ContextWithRelationships(ctx context.Context, typeDescriptors []*config.TypeDescriptor, table string) context.Context {
	typeDescriptor := TypeDescriptorForCurrentTable(typeDescriptors, table)
	relationships := TypeDescriptorRelationships(typeDescriptor)
	return context.WithValue(ctx, config.ContextKey("relationships"), relationships)
}

func TableHasRelationships(cfg *config.Config, table string) bool {
	result := false
	td := TypeDescriptorForCurrentTable(cfg.Descriptor.TypeDescriptors, table)
	if td.TableName == table {
		if TypeDescriptorRelationships(td) != nil {
			result = true
		}
	}
	return result
}

func TypeDescriptorRelationships(td *config.TypeDescriptor) []*config.Field {
	var relationships []*config.Field
	for _, field := range td.Fields {
		if field.Relationship != nil {
			relationships = append(relationships, field)
		}
	}
	return relationships
}

func AppendNoDuplicates(list []map[string]interface{}, item map[string]interface{}) []map[string]interface{} {
	exists := false
	for _, l := range list {
		// TODO: Not performant
		if reflect.DeepEqual(l, item) {
			exists = true
		}
	}
	if !exists {
		return append(list, item)
	}
	return list
}
