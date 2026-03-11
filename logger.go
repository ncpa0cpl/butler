package butler

import (
	"context"
	"fmt"
	"io"
	"log/slog"

	"time"

	"github.com/labstack/gommon/log"
)

type RequestLogger struct {
	logger      ILogger
	request     *Request
	method      string
	path        string
	userHandler LogHandler
}

func newRequestLogger(request *Request, l ILogger) RequestLogger {
	return RequestLogger{l, request, request.Method, request.Path, request.parent.GetReqLogHandler()}
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
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Info, l.request, msg)
	}
	l.logger.Info(msg...)
}

func (l RequestLogger) Infof(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Info, l.request, msg, args)
	}
	l.logger.Infof(msg, args...)
}

func (l RequestLogger) Debug(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Debug, l.request, msg)
	}
	l.logger.Debug(msg...)
}

func (l RequestLogger) Debugf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Debug, l.request, msg, args)
	}
	l.logger.Debugf(msg, args...)
}

func (l RequestLogger) Print(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Print, l.request, msg)
	}
	l.logger.Print(msg...)
}

func (l RequestLogger) Printf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Print, l.request, msg, args)
	}
	l.logger.Printf(msg, args...)
}

func (l RequestLogger) Warn(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Warn, l.request, msg)
	}
	l.logger.Warn(msg...)
}

func (l RequestLogger) Warnf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Warn, l.request, msg, args)
	}
	l.logger.Warnf(msg, args...)
}

func (l RequestLogger) Error(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Error, l.request, msg)
	}
	l.logger.Error(msg...)
}

func (l RequestLogger) Errorf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Error, l.request, msg, args)
	}
	l.logger.Errorf(msg, args...)
}

func (l RequestLogger) Fatal(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Fatal, l.request, msg)
	}
	l.logger.Fatal(msg...)
}

func (l RequestLogger) Fatalf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Fatal, l.request, msg, args)
	}
	l.logger.Fatalf(msg, args...)
}

func (l RequestLogger) Panic(msg ...any) {
	msg = l.addPrefix(msg)
	if l.userHandler != nil {
		msg = l.userHandler.OnLog(LogLevel.Panic, l.request, msg)
	}
	l.logger.Panic(msg...)
}

func (l RequestLogger) Panicf(msg string, args ...any) {
	msg, args = l.addFmtPrefix(msg, args)
	if l.userHandler != nil {
		msg, args = l.userHandler.OnLogf(LogLevel.Panic, l.request, msg, args)
	}
	l.logger.Panicf(msg, args...)
}

type ButlerLogger struct {
	writer     io.Writer
	prefix     string
	lvl        log.Lvl
	timestamps bool
}

func NewButlerLogger(name string, writer io.Writer) *ButlerLogger {
	return &ButlerLogger{
		writer:     writer,
		prefix:     name,
		lvl:        LogLevel.Warn,
		timestamps: true,
	}
}

func (bl *ButlerLogger) PrefixWithTimestamps(timestamps bool) {
	bl.timestamps = timestamps
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
	bl.log(LogLevel.Print, i...)
}

func (bl *ButlerLogger) Printf(format string, args ...any) {
	bl.logf(LogLevel.Print, format, args...)
}

func (bl *ButlerLogger) Debug(i ...any) {
	bl.log(LogLevel.Debug, i...)
}

func (bl *ButlerLogger) Debugf(format string, args ...any) {
	bl.logf(LogLevel.Debug, format, args...)
}

func (bl *ButlerLogger) Info(i ...any) {
	bl.log(LogLevel.Info, i...)
}

func (bl *ButlerLogger) Infof(format string, args ...any) {
	bl.logf(LogLevel.Info, format, args...)
}

func (bl *ButlerLogger) Warn(i ...any) {
	bl.log(LogLevel.Warn, i...)
}

func (bl *ButlerLogger) Warnf(format string, args ...any) {
	bl.logf(LogLevel.Warn, format, args...)
}

func (bl *ButlerLogger) Error(i ...any) {
	bl.log(LogLevel.Error, i...)
}

func (bl *ButlerLogger) Errorf(format string, args ...any) {
	bl.logf(LogLevel.Error, format, args...)
}

func (bl *ButlerLogger) Fatal(i ...any) {
	bl.log(LogLevel.Fatal, i...)
}

func (bl *ButlerLogger) Fatalf(format string, args ...any) {
	bl.logf(LogLevel.Fatal, format, args...)
}

func (bl *ButlerLogger) Panic(i ...any) {
	bl.log(LogLevel.Panic, i...)
}

func (bl *ButlerLogger) Panicf(format string, args ...any) {
	bl.logf(LogLevel.Panic, format, args...)
}

func (bl *ButlerLogger) log(level log.Lvl, args ...any) {
	if level < bl.lvl && level != 0 {
		return
	}

	var message string

	if bl.timestamps {
		message = time.Now().UTC().Format("2006-01-02T15:04:05.999Z07:00") + " "
	}

	if bl.prefix != "" {
		message += bl.prefix + " "
	}

	message += levelString(level)
	message += fmt.Sprint(args...)

	bl.writer.Write([]byte(message + "\n"))
}

func (bl *ButlerLogger) logf(level log.Lvl, format string, args ...any) {
	if level < bl.lvl && level != 0 {
		return
	}

	var message string

	if bl.timestamps {
		message = time.Now().UTC().Format("2006-01-02T15:04:05.999Z07:00") + " "
	}

	if bl.prefix != "" {
		message += bl.prefix + " "
	}

	message += levelString(level)
	message += fmt.Sprintf(format, args...)

	bl.writer.Write([]byte(message + "\n"))
}

type ButlerLoggerSlogHandler struct {
	bl    *ButlerLogger
	attrs []slog.Attr
	name  string
}

func (bl *ButlerLogger) SlogHandler() slog.Handler {
	return &ButlerLoggerSlogHandler{
		bl,
		[]slog.Attr{},
		"",
	}
}

func (h *ButlerLoggerSlogHandler) Enabled(ctx context.Context, lvl slog.Level) bool {
	switch lvl {
	case slog.LevelDebug:
		return h.bl.lvl <= LogLevel.Debug
	case slog.LevelError:
		return h.bl.lvl <= LogLevel.Error
	case slog.LevelWarn:
		return h.bl.lvl <= LogLevel.Warn
	case slog.LevelInfo:
		return h.bl.lvl <= LogLevel.Info
	}
	return false
}

func (h *ButlerLoggerSlogHandler) Handle(ctx context.Context, record slog.Record) error {
	var blvl log.Lvl
	switch record.Level {
	case slog.LevelDebug:
		blvl = LogLevel.Debug
	case slog.LevelError:
		blvl = LogLevel.Error
	case slog.LevelWarn:
		blvl = LogLevel.Warn
	case slog.LevelInfo:
		blvl = LogLevel.Info
	}

	msg := ""

	if h.name != "" {
		msg = fmt.Sprintf("[%s] %s", h.name, record.Message)
	} else {
		msg = record.Message
	}

	record.Attrs(func(a slog.Attr) bool {
		msg = fmt.Sprintf("%s %s=%s", msg, a.Key, a.Value.String())
		return true
	})

	h.bl.log(blvl, "", msg)

	return nil
}

func (h *ButlerLoggerSlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ButlerLoggerSlogHandler{
		h.bl,
		append(h.attrs, attrs...),
		h.name,
	}
}

func (h *ButlerLoggerSlogHandler) WithGroup(name string) slog.Handler {
	return &ButlerLoggerSlogHandler{
		h.bl,
		h.attrs,
		fmt.Sprintf("%s.%s", h.name, name),
	}
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
