// Package for compression HTTP data.
package compression

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"
)

// Allowed compression content types.
const allowedCompressTypes = "text/html application/json"

// Middleware compress writer type.
type compressWriter struct {
	w  http.ResponseWriter
	zw *gzip.Writer
}

// Init new compress writer for HTTP response.
func newCompressWriter(w http.ResponseWriter) *compressWriter {
	return &compressWriter{
		w:  w,
		zw: gzip.NewWriter(w),
	}
}

// Middleware get Header method.
func (c *compressWriter) Header() http.Header {
	return c.w.Header()
}

// Middleware Write method.
func (c *compressWriter) Write(p []byte) (int, error) {
	return c.zw.Write(p)
}

// Middleware WriteHeader method.
func (c *compressWriter) WriteHeader(statusCode int) {
	c.w.Header().Set("Content-Encoding", "gzip")
	c.w.WriteHeader(statusCode)
}

// Close compress stream (writer).
func (c *compressWriter) Close() error {
	return c.zw.Close()
}

// Middleware compression reader type.
type compressReader struct {
	r  io.ReadCloser
	zr *gzip.Reader
}

// Init new compress reader for HTTP requests.
func newCompressReader(r io.ReadCloser) (*compressReader, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}

	return &compressReader{
		r:  r,
		zr: zr,
	}, nil
}

// Middleware Read method.
func (c compressReader) Read(p []byte) (n int, err error) {
	return c.zr.Read(p)
}

// Close compress stream (reader).
func (c *compressReader) Close() error {
	if err := c.r.Close(); err != nil {
		return err
	}
	return c.zr.Close()
}

// Main function for compress/decompress HTTP requests/response.
func GzipCompDecomp(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		ow := w
		supportsContentType := false

		acceptEncoding := r.Header.Get("Accept-Encoding")
		if contentType := r.Header.Get("Content-Type"); contentType != "" {
			supportsContentType = strings.Contains(allowedCompressTypes, contentType)
		}
		supportsGzip := strings.Contains(acceptEncoding, "gzip")
		if supportsGzip && (r.Method == http.MethodGet || supportsContentType) {
			cw := newCompressWriter(w)
			ow = cw
			defer cw.Close()
		}

		contentEncoding := r.Header.Get("Content-Encoding")
		sendsGzip := strings.Contains(contentEncoding, "gzip")
		if sendsGzip {
			cr, err := newCompressReader(r.Body)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			r.Body = cr
			defer cr.Close()
		}

		h.ServeHTTP(ow, r)
	}
}
