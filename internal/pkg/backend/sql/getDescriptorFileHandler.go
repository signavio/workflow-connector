package sql

import (
	"net/http"
	"path/filepath"

	"github.com/signavio/workflow-connector/internal/pkg/config"
	"github.com/signavio/workflow-connector/internal/pkg/log"
	"github.com/spf13/viper"
)

func (b *Backend) GetDescriptorFile(rw http.ResponseWriter, req *http.Request) {
	log.When(config.Options.Logging).Infoln("[request -> http.ServeFile] descriptor file")
	http.ServeFile(rw, req, filepath.Join(viper.Get("configDir").(string), "descriptor.json"))
}
