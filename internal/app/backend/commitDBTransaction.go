package backend

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) CommitTransaction(rw http.ResponseWriter, req *http.Request) {
	requestTx := mux.Vars(req)["commit"]
	routeName := mux.CurrentRoute(req).GetName()
	log.When(config.Options.Logging).Infof("[handler] %s\n", routeName)
	if err := b.CommitTx(requestTx); err != nil {
		if strings.Contains(err.Error(), "404") {
			msg := &util.ResponseMessage{
				Code: http.StatusNotFound,
				Msg: fmt.Sprintf(
					"transaction %s does not exist in the backend's list of open transactions",
					requestTx,
				),
				Tx: requestTx,
			}
			http.Error(rw, msg.String(), http.StatusNotFound)
			return
		}
		http.Error(
			rw,
			err.Error(),
			http.StatusInternalServerError,
		)
		return
	}
	msg := &util.ResponseMessage{
		Code: http.StatusOK,
		Msg: fmt.Sprintf(
			"transaction %s successfully committed to the database",
			requestTx,
		),
		Tx: requestTx,
	}
	rw.Write(msg.Byte())
	return
}
