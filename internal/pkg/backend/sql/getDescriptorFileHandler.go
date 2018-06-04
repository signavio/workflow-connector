package sql

import (
	"context"
	"net/http"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/signavio/workflow-connector/internal/pkg/util"
)

func (b *Backend) GetDescriptorFile(rw http.ResponseWriter, req *http.Request) {
	log.When(config.Options.Logging).Infoln("[request -> http.ServeFile] descriptor file")
	requestWithActiveRoute := req.WithContext(
		context.WithValue(
			req.Context(),
			util.ContextKey("route"),
			"getDescriptorFile"),
	)
	http.ServeFile(rw, requestWithActiveRoute, "./config/descriptor.json")
}
