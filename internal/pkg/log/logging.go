package log

import (
	"fmt"
	"os"
)

type Logger bool

func When(isEnabled bool) Logger {
	if isEnabled {
		return Logger(true)
	}
	return Logger(false)
}
func (l Logger) Infof(format string, v ...interface{}) {
	if l {
		fmt.Printf(format, v...)
	}
}
func (l Logger) Infoln(v ...interface{}) {
	if l {
		fmt.Println(v...)
	}
}
func Fatalln(v ...interface{}) {
	fmt.Println(v...)
	os.Exit(1)
}
func Fatalf(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(1)
}
