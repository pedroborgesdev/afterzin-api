package logger

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	blue   = "\x1b[34m"
	yellow = "\x1b[33m"
	red    = "\x1b[31m"
	green  = "\x1b[32m"
	reset  = "\x1b[0m"
)

func prefix(level string) string {
	var color string
	switch strings.ToUpper(level) {
	case "DEBUG":
		color = blue
	case "INFO":
		color = green
	case "WARNING", "WARN":
		color = yellow
	case "ERROR", "ERR":
		color = red
	default:
		color = reset
	}
	return fmt.Sprintf("[%s%s%s] - %s - ", color, strings.ToUpper(level), reset, time.Now().Format("2006-01-02T15:04:05"))
}

func Debugf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s\n", prefix("DEBUG"), msg)
}

func Infof(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s\n", prefix("INFO"), msg)
}

func Warnf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s\n", prefix("WARNING"), msg)
}

func Errorf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s\n", prefix("ERROR"), msg)
}

func Fatalf(format string, a ...interface{}) {
	Errorf(format, a...)
	os.Exit(1)
}
