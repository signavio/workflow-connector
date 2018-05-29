package sql

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	uuid "github.com/satori/go.uuid"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
)

var txCreatedMsg = func(txUUID string) []byte {
	text := fmt.Sprintf("Transaction %s successfully added to backend", txUUID)
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

func (b *Backend) CreateDBTransaction(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options).Infof("[handler] %s\n", routeName)
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
	log.When(config.Options).Infof("[handler] Added transaction %s to backend\n", txUUID)
	// Explicitly call cancel after delay
	go func(c context.CancelFunc, d time.Duration, id uuid.UUID) {
		select {
		case <-time.After(d):
			c()
			_, ok := b.Transactions.Load(fmt.Sprintf("%s", id))
			if ok {
				b.Transactions.Delete(id)
				log.When(config.Options).Infof("[handler] Timeout expired: \n"+
					"Open transaction %s has been deleted from backend\n", id)
			}
		}
	}(cancel, delay, txUUID)
	rw.Write(txCreatedMsg(fmt.Sprintf("%s", txUUID)))
	return
}
