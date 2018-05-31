package log

import (
	"fmt"
)

type Logger bool

func When(isEnabled bool) Logger {
	if isEnabled {
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
