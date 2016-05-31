// Packager logger ...
package logger

type logLevel uint8

const (
	DEBUG = iota
	TRACE
	INFO
	WARN
	ERROR
)

type logRecord struct {
	ts      int64
	content string
	level   logLevel
}

type Logwriter interface {
	WriteLog(record *logRecord)
}

var gLog *Logger

type Logger struct {
	w     Logwriter
	level logLevel
}

func (l *Logger) Debug(args ...interface{}) {
	l.w.WriteLog(record)
}
