package gateway

import (
	"bytes"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SlowClientBuffer struct {
	maxBodySize int64
	pool        sync.Pool
}

func NewSlowClientBuffer(maxBodySize int64) *SlowClientBuffer {
	return &SlowClientBuffer{
		maxBodySize: maxBodySize,
		pool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (scb *SlowClientBuffer) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body == nil {
			c.Next()
			return
		}

		contentLength := c.Request.ContentLength
		if contentLength > 0 && contentLength > scb.maxBodySize {
			buf := scb.pool.Get().(*bytes.Buffer)
			buf.Reset()
			defer scb.pool.Put(buf)

			if _, err := io.CopyBuffer(buf, c.Request.Body, make([]byte, 8*1024)); err != nil {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "Failed to read request body",
				})
				return
			}

			c.Request.Body = io.NopCloser(buf)
		}

		c.Next()
	}
}

type RequestBuffer struct {
	mu            sync.RWMutex
	requestCount  int64
	bytesRead     int64
	maxBufferSize int64
}

func NewRequestBuffer(maxBufferSize int64) *RequestBuffer {
	return &RequestBuffer{
		maxBufferSize: maxBufferSize,
	}
}

func (rb *RequestBuffer) WrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && rb.maxBufferSize > 0 {
			body := r.Body
			defer body.Close()

			var buf bytes.Buffer
			limitedReader := io.LimitReader(body, rb.maxBufferSize)

			n, err := io.Copy(&buf, limitedReader)
			if err == nil {
				rb.mu.Lock()
				rb.requestCount++
				rb.bytesRead += n
				rb.mu.Unlock()

				r.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
			} else if err == io.EOF {
				rb.mu.Lock()
				rb.requestCount++
				rb.bytesRead += n
				rb.mu.Unlock()

				r.Body = io.NopCloser(bytes.NewReader(buf.Bytes()))
			} else {
				http.Error(w, "Request body too large", http.StatusRequestEntityTooLarge)
				return
			}

			if buf.Len() >= int(rb.maxBufferSize) {
				remaining, err := io.Copy(io.Discard, body)
				if err == nil {
					rb.mu.Lock()
					rb.bytesRead += remaining
					rb.mu.Unlock()
				}
			}
		}

		handler.ServeHTTP(w, r)
	})
}

func (rb *RequestBuffer) Stats() (int64, int64) {
	rb.mu.RLock()
	defer rb.mu.RUnlock()
	return rb.requestCount, rb.bytesRead
}

type SlowClientHandler struct {
	readTimeout  time.Duration
	writeTimeout time.Duration
	bufferSize   int64
}

func NewSlowClientHandler(readTimeout, writeTimeout time.Duration, bufferSize int64) *SlowClientHandler {
	return &SlowClientHandler{
		readTimeout:  readTimeout,
		writeTimeout: writeTimeout,
		bufferSize:   bufferSize,
	}
}

func (sch *SlowClientHandler) WrapHandler(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if sch.readTimeout > 0 {
			wrapped := &timeoutWriter{
				ResponseWriter: w,
				timeout:        sch.readTimeout,
			}
			handler.ServeHTTP(wrapped, r)
		} else {
			handler.ServeHTTP(w, r)
		}
	})
}

type timeoutWriter struct {
	http.ResponseWriter
	timeout time.Duration
}

func (tw *timeoutWriter) Write(p []byte) (int, error) {
	return tw.ResponseWriter.Write(p)
}
