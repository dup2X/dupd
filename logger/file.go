// Package logger ...
package logger

type fileWriter struct {
	in chan *logRecord
}

func (f *fileWriter) WriterLog(record *logRecord) {
	select {
	case f.in <- record:
	default:
		println("full")
	}
}
