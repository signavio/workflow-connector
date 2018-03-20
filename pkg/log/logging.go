package log

import (
	"fmt"

	"github.com/signavio/workflow-connector/pkg/config"
)

type Logger bool

func When(cfg *config.Config) Logger {
	if cfg.Logging != "" {
		return Logger(true)
	}
	return Logger(false)
}

func (l Logger) Infof(format string, args ...interface{}) {
	if l {
		fmt.Printf(format, args...)
	}
}

func (l Logger) Infoln(args ...interface{}) {
	if l {
		fmt.Println(args...)
	}
}
