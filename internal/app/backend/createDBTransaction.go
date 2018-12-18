package backend

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) CreateTransaction(rw http.ResponseWriter, req *http.Request) {
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)
	txUUID, err := b.CreateTx(60 * time.Second)
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
			Tx:   txUUID.String(),
		}
		http.Error(rw, msg.String(), http.StatusInternalServerError)
		return
	}
	msg := &util.ResponseMessage{
		Code: http.StatusInternalServerError,
		Msg: fmt.Sprintf(
			"transaction %s successfully added to backend",
			txUUID,
		),
		Tx: txUUID.String(),
	}
	rw.Write(msg.Byte())
	return
}
