package util

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

	"github.com/signavio/workflow-connector/internal/pkg/config"
)

// ContextKey is used as a key when populating a context.Context with values
type ContextKey string

var (
	ErrCardinalityMany = errors.New("Form data contained multiple input values for a single column")
	ErrUnexpectedJSON  = errors.New("Received JSON data that we are unable to parse")
)

// GetTypeDescriptorUsingDBTableName will return the typeDescriptor from the descriptor.json
// file defined for the table provided in the function's second parameter
func GetTypeDescriptorUsingDBTableName(typeDescriptors []*config.TypeDescriptor, tableName string) (td *config.TypeDescriptor) {
	for _, typeDescriptor := range typeDescriptors {
		if tableName == typeDescriptor.TableName {
			td = typeDescriptor
		}
	}
	return
}

// GetDBTableNameUsingTypeDescriptorKey will return the typeDescriptor from the descriptor.json file defined for the table provided in the function's second parameter
func GetDBTableNameUsingTypeDescriptorKey(typeDescriptors []*config.TypeDescriptor, typeDescriptorKey string) (tableName string) {
	for _, typeDescriptor := range typeDescriptors {
		if typeDescriptorKey == typeDescriptor.Key {
			tableName = typeDescriptor.TableName
		}
	}
	return
}

func GetTypeDescriptorUsingTypeDescriptorKey(typeDescriptors []*config.TypeDescriptor, typeDescriptorKey string) (result *config.TypeDescriptor) {
	for _, typeDescriptor := range typeDescriptors {
		if typeDescriptorKey == typeDescriptor.Key {
			result = typeDescriptor
		}
	}
	return
}

// ContextWithRelationships will return a new context which will included an array of all relationships for the table provided in the function's second parameter
func ContextWithRelationships(ctx context.Context, typeDescriptors []*config.TypeDescriptor, table string) context.Context {
	typeDescriptor := GetTypeDescriptorUsingDBTableName(typeDescriptors, table)
	relationships := TypeDescriptorRelationships(typeDescriptor)
	return context.WithValue(ctx, ContextKey("relationships"), relationships)
}

func TableHasRelationships(cfg config.Config, table string) bool {
	result := false
	td := GetTypeDescriptorUsingDBTableName(cfg.Descriptor.TypeDescriptors, table)
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

func ParseDataForm(req *http.Request) (data map[string]interface{}, err error) {
	switch req.Header.Get("Content-Type") {
	case "application/json":
		return parseApplicationJSON(req)
	default:
		return parseFormURLEncoded(req)
	}
}

func parseFormURLEncoded(req *http.Request) (data map[string]interface{}, err error) {

	if err := req.ParseForm(); err != nil {
		return nil, err
	}
	if len(req.Form) == 0 {
		body, _ := ioutil.ReadAll(req.Body)
		defer req.Body.Close()
		return nil, fmt.Errorf("Unable to parse the request body:\n%s\n", body)
	}
	data = make(map[string]interface{})
	for k, v := range req.Form {
		if len(v) > 1 {
			return nil, ErrCardinalityMany
		}
		data[k] = v[0]
	}
	fmt.Printf("DEBUG: %+v\n", req.Form) // output for debug
	return data, nil
}

func parseApplicationJSON(req *http.Request) (data map[string]interface{}, err error) {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	defer req.Body.Close()
	data = make(map[string]interface{})
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf(ErrUnexpectedJSON.Error()+": %v\n", err)
	}
	return
}
