package sql

import (
	"fmt"

	"github.com/sdaros/workflow-db-connector/pkg/config"
	"github.com/sdaros/workflow-db-connector/pkg/log"
	"github.com/sdaros/workflow-db-connector/pkg/util"
)

func (r *getSingle) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handlers] getSingle")
	r.table = r.ctx.Value(config.ContextKey("table")).(string)
	// Use the TableSchema containing columns of related tables if the
	// current table contains 1..* relationship with other tables
	if util.TableHasRelationships(r.backend.Cfg, r.table) {
		r.columnNames = r.backend.Cfg.TableSchemas[fmt.Sprintf("%s_relationships", r.table)].ColumnNames
		r.dataTypes = r.backend.Cfg.TableSchemas[fmt.Sprintf("%s_relationships", r.table)].DataTypes
	} else {
		r.columnNames = r.backend.Cfg.TableSchemas[r.table].ColumnNames
		r.dataTypes = r.backend.Cfg.TableSchemas[r.table].DataTypes
	}
	queryText, err := r.backend.interpolateGetTemplate(
		r.ctx,
		r.backend.Templates["GetSingleWithRelationships"],
		r.table)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[endpoint <- template] Interpolated `GetSingleWithRelationships`:\n%s\n",
		queryText,
	)
	results, err = r.getQueryResults(r.ctx, queryText, r.id)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[endpoint <- db] getQueryResults: \n%+v\n",
		results,
	)
	return
}

func (r *getCollection) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handlers] getCollection")
	table := r.ctx.Value(config.ContextKey("table")).(string)
	r.columnNames = r.backend.Cfg.TableSchemas[table].ColumnNames
	r.dataTypes = r.backend.Cfg.TableSchemas[table].DataTypes
	queryText := fmt.Sprintf(r.backend.Queries["GetCollection"], table)
	results, err = r.getQueryResults(r.ctx, queryText)
	if err != nil {
		return nil, err
	}
	if len(results) > 2 {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults (head): \n%+v ...\n",
			results,
		)
	}
	return
}

func (r *getSingleAsOption) handle() (results []interface{}, err error) {
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query, r.id)
	log.When(r.backend.Cfg).Infof(
		"[endpoint <- db] getQueryResults: \n%+v\n",
		results,
	)
	return
}

func (r *getCollectionAsOptions) handle() (results []interface{}, err error) {
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query)
	if len(results) > 2 {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults (head): \n%+v ...\n",
			results,
		)
	}
	return
}

func (r *getCollectionAsOptionsFilterable) handle() (results []interface{}, err error) {
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query, r.filter)
	if len(results) > 2 {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[endpoint <- db] getQueryResults (head): \n%+v ...\n",
			results,
		)
	}
	return
}

func columnNamesAndDataTypesForOptionRoutes(b *Backend, table, columnAsOptionName string) (columnNames []string, dataTypes []interface{}) {
	columnNamesAndDataTypes := make(map[string]interface{})
	for i, columnName := range b.Cfg.TableSchemas[table].ColumnNames {
		columnNamesAndDataTypes[columnName] = b.Cfg.TableSchemas[table].DataTypes[i]
	}
	IDName := []string{
		fmt.Sprintf("%s_%s", table, "id"),
		fmt.Sprintf("%s_%s", table, "name"),
	}
	IDNameDataTypes := []interface{}{
		columnNamesAndDataTypes[fmt.Sprintf("%s_%s", table, "id")],
		columnNamesAndDataTypes[fmt.Sprintf("%s_%s", table, columnAsOptionName)],
	}
	return IDName, IDNameDataTypes
}

func (r *updateSingle) handle() (results []interface{}, err error) {
	r.backend.RequestData, err = parseDataForm(r.request)
	if err != nil {
		return nil, err
	}
	updateSingle, err := r.backend.interpolateTemplate(
		r.request.Context(),
		r.backend.Templates["UpdateSingle"],
	)
	if err != nil {
		return nil, err
	}
	args := r.backend.buildExecQueryArgsWithID(r.request.Context(), r.id)
	return r.backend.execContext(r.request.Context(), updateSingle, args)
}

func (r *createSingle) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handlers] createSingle")
	r.backend.RequestData, err = parseDataForm(r.request)
	if err != nil {
		return nil, err
	}
	createSingle, err := r.backend.interpolateTemplate(
		r.request.Context(), r.backend.Templates["CreateSingle"])
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[endpoint <- template] Interpolated `CreateSingle`:\n%s\n",
		createSingle,
	)

	args := r.backend.buildExecQueryArgs(r.request.Context())
	results, err = r.backend.execContext(r.request.Context(), createSingle, args)
	log.When(r.backend.Cfg).Infof(
		"[endpoint <- db] execContext: \nresults: %+v\nerror: %+v\n",
		results,
		err,
	)
	return
}
