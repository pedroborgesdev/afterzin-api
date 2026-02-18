package logger

import (
	"fmt"
	"os"
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
	return fmt.Sprintf("[%s] %s ", level, time.Now().Format("2006-01-02T15:04:05"))
}

func Debugf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s%s%s\n", prefix("DEBUG"), blue, msg, reset)
}

func Infof(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s%s%s\n", prefix("INFO"), green, msg, reset)
}

func Warnf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s%s%s\n", prefix("WARNING"), yellow, msg, reset)
}

func Errorf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	fmt.Printf("%s%s%s%s\n", prefix("ERROR"), red, msg, reset)
}

func Fatalf(format string, a ...interface{}) {
	Errorf(format, a...)
	os.Exit(1)
}
