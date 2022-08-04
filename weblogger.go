package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"go.elastic.co/apm/v2"
)

const (
	logTimeFormat     = "02/Jan/2006 15:04:05" // required time format for sfagent to pick up
	logFilePath       = "C:\\Users\\Sonika.Prakash\\GitHub\\goji web app\\web.log"
	logFormat         = "[%s] | elasticapm transaction.id=%s trace.id=%s span.id=%s\n"
	infoPrefixFormat  = "[%s] [info] "
	debugPrefixFormat = "[%s] [debug] "
	errorPrefixFormat = "[%s] [error] "
)

// logging levels
var (
	Info  *log.Logger
	Debug *log.Logger
	Error *log.Logger
	// Warn *log.Logger
)

// CtxLabels stores current transaction IDs
type CtxLabels struct {
	transactionID string
	traceID       string
	spanID        string
	curCtx        context.Context
}

var ctxLabel CtxLabels

// LogWriter is the custom log writer
type LogWriter struct {
	bw     *bufio.Writer
	f      *os.File
	format string
}

// NewLogWriter opens the log file and sets the logWriter struct
func NewLogWriter(fname, format string) *LogWriter {
	file, _ := os.OpenFile(fname, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	return &LogWriter{bw: bufio.NewWriter(file), f: file, format: format}
}

func (flw *LogWriter) Write(bs []byte) (int, error) {
	defer flw.bw.Flush()
	// return flw.bw.WriteString(flw.format + string(bs))
	logStr := fmt.Sprintf(flw.format, time.Now().UTC().Format(logTimeFormat)) + logFormat
	msg := strings.TrimRight(string(bs), "\r\n")
	ctxLabel.getTraceLabels(ctxLabel.curCtx)
	return flw.bw.WriteString(fmt.Sprintf(logStr, msg, ctxLabel.transactionID, ctxLabel.traceID, ctxLabel.spanID))
}

// getTraceLabels gets the transaction, trace, and span IDs from the context passed
func (c *CtxLabels) getTraceLabels(ctx context.Context) {
	tx := apm.TransactionFromContext(ctx)
	if tx != nil {
		traceContext := tx.TraceContext()
		c.transactionID = traceContext.Span.String()
		c.traceID = traceContext.Trace.String()
		if span := apm.SpanFromContext(ctx); span != nil {
			c.spanID = span.TraceContext().Span.String()
		} else {
			c.spanID = "None"
		}
	}
}

func init() {
	Info = log.New(NewLogWriter(logFilePath, infoPrefixFormat), "", 0)
	Debug = log.New(NewLogWriter(logFilePath, debugPrefixFormat), "", 0)
	Error = log.New(NewLogWriter(logFilePath, errorPrefixFormat), "", 0)
}
