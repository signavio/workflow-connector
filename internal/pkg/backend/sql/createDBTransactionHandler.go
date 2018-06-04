package sql

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

var (
	txCreatedMsg = func(code int, txUUID uuid.UUID) fmt.Stringer {
		msg := &util.HTTPCodeTxMsg{
			code,
			txUUID.String(),
			fmt.Sprintf(
				"transaction %s successfully added to backend",
				txUUID,
			),
		}
		return msg
	}
)

func (b *Backend) CreateDBTransaction(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)
	delay := 60 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), delay)
	tx, err := b.DB.BeginTx(ctx, nil)
	if err != nil {
		cancel()
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	txUUID := uuid.NewV4()
	if err != nil {
		cancel()
		http.Error(rw, err.Error(), http.StatusInternalServerError)
		return
	}
	b.Transactions.Store(fmt.Sprintf("%s", txUUID), tx)
	log.When(config.Options.Logging).Infof("[handler] added transaction %s to backend\n", txUUID)
	// Explicitly call cancel after delay
	go func(c context.CancelFunc, d time.Duration, id uuid.UUID) {
		select {
		case <-time.After(d):
			c()
			_, ok := b.Transactions.Load(fmt.Sprintf("%s", id))
			if ok {
				b.Transactions.Delete(id)
				log.When(config.Options.Logging).Infof("[handler] timeout expired: \n"+
					"transaction %s has been deleted from backend\n", id)
			}
		}
	}(cancel, delay, txUUID)
	rw.Write([]byte(txCreatedMsg(http.StatusOK, txUUID).String()))
	return
}
