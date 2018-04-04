package sql

import (
	"database/sql"
	"fmt"
	"errors"

	"github.com/signavio/workflow-connector/pkg/config"
	"github.com/signavio/workflow-connector/pkg/log"
	"github.com/signavio/workflow-connector/pkg/util"
)

var ErrNoLastInsertID = errors.New("Database does not support getting the last inserted ID")

func (r *getSingle) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] getSingle")
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
		"[handler <- template] Interpolated `GetSingleWithRelationships`:\n%s\n",
		queryText,
	)
	results, err = r.getQueryResults(r.ctx, queryText, r.id)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] getQueryResults: \n%+v\n",
		results,
	)
	return
}

func (r *getCollection) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] getCollection")
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
			"[handler <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[handler <- db] getQueryResults (head): \n%+v ...\n",
			results,
		)
	}
	return
}

func (r *getSingleAsOption) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] getSingleAsOptions")
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query, r.id)
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] getQueryResults: \n%+v\n",
		results,
	)
	return
}

func (r *getCollectionAsOptions) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] getCollectionAsOptions")
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query)
	if len(results) > 2 {
		log.When(r.backend.Cfg).Infof(
			"[handler <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[handler <- db] getQueryResults (head): \n%+v ...\n",
			results,
		)
	}
	return
}

func (r *getCollectionAsOptionsFilterable) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] getCollectionAsOptionsFilterable")
	currentTable := r.ctx.Value(config.ContextKey("table")).(string)
	columnAsOptionName := r.ctx.Value(config.ContextKey("columnAsOptionName")).(string)
	r.columnNames, r.dataTypes = columnNamesAndDataTypesForOptionRoutes(
		r.backend, currentTable, columnAsOptionName,
	)
	results, err = r.getQueryResults(r.ctx, r.query, r.filter)
	if len(results) > 2 {
		log.When(r.backend.Cfg).Infof(
			"[handler <- db] getQueryResults: \n%+v ...\n",
			results[0:1],
		)
	} else {
		log.When(r.backend.Cfg).Infof(
			"[handler <- db] getQueryResults (head): \n%+v ...\n",
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
	log.When(r.backend.Cfg).Infoln("[handler] updateSingle")
	r.backend.RequestData, err = parseDataForm(r.request)
	if err != nil {
		return nil, err
	}
	updateSingle, err := r.backend.interpolateTemplate(
		r.request.Context(), r.backend.Templates["UpdateSingle"],
	)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[handler <- template] Interpolated `CreateSingle`:\n%s\n",
		updateSingle,
	)
	args := r.backend.buildExecQueryArgsWithID(r.request.Context(), r.id)
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] buildExecQueryArgsWithID(): returned following args:\n%s\n",
		args,
	)
	result, err := r.backend.execContext(r.request.Context(), updateSingle, args)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[handler <-> handlers] return the newly updated resource:\n call getSingle.handle()",
		result,
		err,
	)
	route := &getSingle{
		ctx:     r.request.Context(),
		id:      r.id,
		backend: r.backend,
	}
	results, err = route.handle()
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] get just updated resource: \nresults: %+v\nerror: %+v\n",
		results,
		err,
	)
	return
}

func (r *createSingle) handle() (results []interface{}, err error) {
	log.When(r.backend.Cfg).Infoln("[handler] createSingle")
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
		"[handler <- template] Interpolated `CreateSingle`:\n%s\n",
		createSingle,
	)
	args := r.backend.buildExecQueryArgs(r.request.Context())
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] buildExecQueryArgs(): returned following args:\n%s\n",
		args,
	)
	result, err := r.backend.execContext(r.request.Context(), createSingle, args)
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infoln("[handler <-> handlers] return the " +
		"newly created resource:\n call getSingle.handle()",
	)
	return r.getJustCreated(result)
}
func (r *createSingle) getJustCreated(result sql.Result) (results []interface{}, err error) {
	// TODO: Figure out how to handle result.RowsAffected()
	id, err := result.LastInsertId()
	if err != nil {
		// LastInsertID() not supported, return only a 200 http.StatusCode
		return []interface{}{}, nil
	}
	if id < 1 {
		// getting the last inserted id probably not supported
		return nil, nil
	}
	route := &getSingle{
		ctx:     r.request.Context(),
		id:      fmt.Sprintf("%d", id),
		backend: r.backend,
	}
	results, err = route.handle()
	if err != nil {
		return nil, err
	}
	log.When(r.backend.Cfg).Infof(
		"[handler <- db] getJustCreated(): \nresults: %+v\nerror: %+v\n",
		results,
		err,
	)
	return

}
