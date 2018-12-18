package backend

import (
	"encoding/json"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetDescriptorFile(rw http.ResponseWriter, req *http.Request) {
	log.When(config.Options.Logging).Infoln("[request -> http.ServeFile] descriptor file")
	descriptor, err := json.MarshalIndent(&config.Options.Descriptor, "", "  ")
	if err != nil {
		msg := &util.ResponseMessage{
			Code: http.StatusInternalServerError,
			Msg:  err.Error(),
		}
		http.Error(rw, msg.Error(), http.StatusInternalServerError)
		return
	}
	rw.Write(descriptor)
}
