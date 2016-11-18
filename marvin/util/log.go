package util

import (
	"fmt"
	"os"

	"github.com/mgutz/ansi"
)

var (
	funcErr   = ansi.ColorFunc("red+h")
	funcWarn  = ansi.ColorFunc("yellow")
	funcGood  = ansi.ColorFunc("green")
	funcDebug = ansi.ColorFunc("black+h")
)

func LogIfError(err error) error {
	if err != nil {
		LogError(err)
	}
	return err
}

func LogError(err error) {
	fmt.Fprintln(os.Stderr, funcErr(fmt.Sprintf("[  ERR] %+v\n", err)))
}

func LogBad(msg ...interface{}) {
	msg = append([]interface{}{"[  ERR]"}, msg...)
	fmt.Fprint(os.Stderr, funcErr(fmt.Sprintln(msg...)))
}

func LogBadf(f string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, funcErr(fmt.Sprintf("%s%s", "[  ERR]", fmt.Sprintf(f, v...))))
}

func LogWarn(msg ...interface{}) {
	msg = append([]interface{}{"[ WARN]"}, msg...)
	fmt.Fprint(os.Stderr, funcWarn(fmt.Sprintln(msg...)))
}

func LogWarnf(f string, v ...interface{}) {
	fmt.Fprintln(os.Stderr, funcWarn(fmt.Sprintf("%s%s", "[ WARN]", fmt.Sprintf(f, v...))))
}

func LogDebug(msg ...interface{}) {
	msg = append([]interface{}{"[DEBUG]"}, msg...)
	fmt.Fprint(os.Stderr, funcDebug(fmt.Sprintln(msg...)))
}

func LogGood(msg ...interface{}) {
	msg = append([]interface{}{"[ INFO]"}, msg...)
	fmt.Fprint(os.Stderr, funcGood(fmt.Sprintln(msg...)))
}
