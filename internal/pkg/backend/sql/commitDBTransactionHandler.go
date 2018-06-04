package sql

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	txCommittedMsg = func(code int, txUUID string) fmt.Stringer {
		msg := &util.HTTPCodeTxMsg{
			code,
			txUUID,
			fmt.Sprintf(
				"transaction %s successfully committed to the database",
				txUUID,
			),
		}
		return msg
	}
	errTxUUIDInvalid = func(code int, txUUID string) fmt.Stringer {
		msg := &util.HTTPErrorCodeMsg{
			code,
			fmt.Sprintf(
				"transaction %s does not exist in the backend's list of open transactions",
				txUUID,
			),
		}
		return msg
	}
)

func (b *Backend) CommitDBTransaction(rw http.ResponseWriter, req *http.Request) {
	requestTx := mux.Vars(req)["commit"]
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)

	tx, ok := b.Transactions.Load(requestTx)
	if !ok {
		http.Error(
			rw,
			errTxUUIDInvalid(http.StatusNotFound, requestTx).String(),
			http.StatusNotFound,
		)
		return
	}
	if err := tx.(*sql.Tx).Commit(); err != nil {
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	b.Transactions.Delete(requestTx)
	rw.Write([]byte(txCommittedMsg(http.StatusOK, requestTx).String()))
	return
}
