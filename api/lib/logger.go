package lib

import (
	"bytes"
	"log"
	"net/http"
)

type LoggerWrapper struct {
	h      http.HandlerFunc
	logger *log.Logger
}

type InterceptedHttpResponseWriter struct {
	status int
	buff   bytes.Buffer
	writer http.ResponseWriter
}

func (i *InterceptedHttpResponseWriter) Header() http.Header {
	return i.writer.Header()
}

func (i *InterceptedHttpResponseWriter) Write(b []byte) (int, error) {
	i.buff.Write(b)
	return i.writer.Write(b)
}

func (i *InterceptedHttpResponseWriter) WriteHeader(statusCode int) {
	i.status = statusCode
	i.writer.WriteHeader(statusCode)
}

func (l LoggerWrapper) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	l.logger.Printf("Request %q from: %s", r.URL, r.RemoteAddr)
	i := InterceptedHttpResponseWriter{
		writer: w,
		status: http.StatusOK,
	}
	l.h(&i, r)
	l.logger.Printf("Response %q, status %d to: %s\n", i.buff.String(), i.status, r.RemoteAddr)
}
