package butler

import (
	"encoding/json"
	"fmt"
	"io"

	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

type RequestLogger struct {
	logger echo.Logger
	method string
	path   string
}

func newRequestLogger(method, path string, l echo.Logger) RequestLogger {
	return RequestLogger{l, method, path}
}

func (l RequestLogger) addPrefix(msg []any) []any {
	msg = append([]any{fmt.Sprintf("%s %s - ", l.method, l.path)}, msg...)
	return msg
}

func (l RequestLogger) addFmtPrefix(msg string, args []any) (string, []any) {
	args = append([]any{l.method, l.path}, args...)
	return "%s %s - " + msg, args
}

func (l RequestLogger) Info(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Info(msg...)
}

func (l RequestLogger) Infof(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Infof(msg, args...)
}

func (l RequestLogger) Debug(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Debug(msg...)
}

func (l RequestLogger) Debugf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Debugf(msg, args...)
}

func (l RequestLogger) Print(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Print(msg...)
}

func (l RequestLogger) Printf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Printf(msg, args...)
}

func (l RequestLogger) Warn(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Warn(msg...)
}

func (l RequestLogger) Warnf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Warnf(msg, args...)
}

func (l RequestLogger) Error(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Error(msg...)
}

func (l RequestLogger) Errorf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Errorf(msg, args...)
}

func (l RequestLogger) Fatal(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Fatal(msg...)
}

func (l RequestLogger) Fatalf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Fatalf(msg, args...)
}

func (l RequestLogger) Panic(msg ...any) {
	msg = l.addPrefix(msg)
	l.logger.Panic(msg...)
}

func (l RequestLogger) Panicf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	l.logger.Panicf(msg, args...)
}

type ButlerLogger struct {
	writer io.Writer
	prefix string
	lvl    log.Lvl
}

func NewButlerLogger(name string, writer io.Writer) *ButlerLogger {
	return &ButlerLogger{
		writer: writer,
		prefix: name,
		lvl:    LogLevel.Warn,
	}
}

func (bl *ButlerLogger) Output() io.Writer {
	return bl.writer
}

func (bl *ButlerLogger) SetOutput(w io.Writer) {
	bl.writer = w
}

func (bl *ButlerLogger) Prefix() string {
	return bl.prefix
}

func (bl *ButlerLogger) SetPrefix(p string) {
	bl.prefix = p
}

func (bl *ButlerLogger) Level() log.Lvl {
	return bl.lvl
}

func (bl *ButlerLogger) SetLevel(v log.Lvl) {
	bl.lvl = v
}

func (bl *ButlerLogger) SetHeader(h string) {}

func (bl *ButlerLogger) Print(i ...any) {
	bl.log(LogLevel.Print, "", i...)
}

func (bl *ButlerLogger) Printf(format string, args ...any) {
	bl.log(LogLevel.Print, format, args...)
}

func (bl *ButlerLogger) Printj(j log.JSON) {
	bl.log(LogLevel.Print, "type:json", j)
}

func (bl *ButlerLogger) Debug(i ...any) {
	bl.log(LogLevel.Debug, "", i...)
}

func (bl *ButlerLogger) Debugf(format string, args ...any) {
	bl.log(LogLevel.Debug, format, args...)
}

func (bl *ButlerLogger) Debugj(j log.JSON) {
	bl.log(LogLevel.Debug, "type:json", j)
}

func (bl *ButlerLogger) Info(i ...any) {
	bl.log(LogLevel.Info, "", i...)
}

func (bl *ButlerLogger) Infof(format string, args ...any) {
	bl.log(LogLevel.Info, format, args...)
}

func (bl *ButlerLogger) Infoj(j log.JSON) {
	bl.log(LogLevel.Info, "type:json", j)
}

func (bl *ButlerLogger) Warn(i ...any) {
	bl.log(LogLevel.Warn, "", i...)
}

func (bl *ButlerLogger) Warnf(format string, args ...any) {
	bl.log(LogLevel.Warn, format, args...)
}

func (bl *ButlerLogger) Warnj(j log.JSON) {
	bl.log(LogLevel.Warn, "type:json", j)
}

func (bl *ButlerLogger) Error(i ...any) {
	bl.log(LogLevel.Error, "", i...)
}

func (bl *ButlerLogger) Errorf(format string, args ...any) {
	bl.log(LogLevel.Error, format, args...)
}

func (bl *ButlerLogger) Errorj(j log.JSON) {
	bl.log(LogLevel.Error, "type:json", j)
}

func (bl *ButlerLogger) Fatal(i ...any) {
	bl.log(LogLevel.Fatal, "", i...)
}

func (bl *ButlerLogger) Fatalj(j log.JSON) {
	bl.log(LogLevel.Fatal, "type:json", j)
}

func (bl *ButlerLogger) Fatalf(format string, args ...any) {
	bl.log(LogLevel.Fatal, format, args...)
}

func (bl *ButlerLogger) Panic(i ...any) {
	bl.log(LogLevel.Panic, "", i...)
}

func (bl *ButlerLogger) Panicj(j log.JSON) {
	bl.log(LogLevel.Panic, "type:json", j)
}

func (bl *ButlerLogger) Panicf(format string, args ...any) {
	bl.log(LogLevel.Panic, format, args...)
}

func (bl *ButlerLogger) log(level log.Lvl, format string, args ...any) {
	if level < bl.lvl && level != 0 {
		return
	}

	var message string
	if format == "" {
		message = fmt.Sprint(args...)
	} else if format == "json" {
		b, err := json.Marshal(args[0])
		if err != nil {
			panic(err)
		}
		message = string(b)
	} else {
		message = fmt.Sprintf(format, args...)
	}

	t := time.Now().UTC()
	bl.writer.Write([]byte(
		t.Format("2006-01-02T15:04:05.999Z07:00") +
			" " +
			levelString(level) +
			message +
			"\n",
	))
}

func levelString(level log.Lvl) string {
	switch level {
	case LogLevel.Debug:
		return "DEBUG: "
	case LogLevel.Error:
		return "ERROR: "
	case LogLevel.Fatal:
		return "FATAL: "
	case LogLevel.Info:
		return "INFO: "
	case LogLevel.Panic:
		return "PANIC: "
	case LogLevel.Warn:
		return "WARN: "
	case LogLevel.Print:
		return ""
	}

	return ""
}

type tlvl struct {
	Print log.Lvl
	Info  log.Lvl
	Debug log.Lvl
	Warn  log.Lvl
	Error log.Lvl
	Fatal log.Lvl
	Panic log.Lvl
}

var LogLevel = tlvl{
	Print: 0,
	Debug: 1,
	Info:  2,
	Warn:  3,
	Error: 4,
	Fatal: 5,
	Panic: 6,
}
