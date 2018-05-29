package sql

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
)

var errTxUUIDInvalid = func(txUUID string) string {
	text := fmt.Sprintf(
		"Transaction %s does not exist in the backend's list of open transactions",
		txUUID,
	)
	msg := map[string]interface{}{
		"status": map[string]string{
			"code":        "400",
			"description": text,
		},
		"transactionUUID": txUUID,
	}
	result, _ := json.MarshalIndent(&msg, "", "  ")
	return string(result[:])
}
var txCommittedMsg = func(txUUID string) []byte {
	text := fmt.Sprintf(
		"Transaction %s successfully committed to the database",
		txUUID,
	)
	msg := map[string]interface{}{
		"status": map[string]string{
			"code":        "200",
			"description": text,
		},
		"transactionUUID": txUUID,
	}
	result, _ := json.MarshalIndent(&msg, "", "  ")
	return result
}

func (b *Backend) CommitDBTransaction(rw http.ResponseWriter, req *http.Request) {
	requestTx := mux.Vars(req)["commit"]
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options).Infof("[handler] %s\n", routeName)

	tx, ok := b.Transactions.Load(requestTx)
	if !ok {
		http.Error(rw, errTxUUIDInvalid(requestTx), http.StatusBadRequest)
		return
	}
	if err := tx.(*sql.Tx).Commit(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	b.Transactions.Delete(requestTx)
	rw.Write(txCommittedMsg(requestTx))
	return
}
