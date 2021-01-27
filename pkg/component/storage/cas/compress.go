package cas

import (
	"compress/gzip"
	"io"
)

// GzipWriter .
type GzipWriter struct {
	closer io.WriteCloser
	writer *gzip.Writer
}

// NewGzipWriter .
func NewGzipWriter(closer io.WriteCloser) io.WriteCloser {
	gzipw := gzip.NewWriter(closer)
	gzipw.Header.Name = ""

	gw := GzipWriter{
		closer: closer,
		writer: gzipw,
	}

	return &gw
}

// Write .
func (gw *GzipWriter) Write(p []byte) (int, error) {
	return gw.writer.Write(p)
}

// Close .
func (gw *GzipWriter) Close() error {
	defer gw.writer.Close()
	return gw.closer.Close()
}

// GzipReader .
type GzipReader struct {
	closer io.ReadCloser
	reader *gzip.Reader
}

// NewGzipReader .
func NewGzipReader(closer io.ReadCloser) (io.ReadCloser, error) {
	gzipr, err := gzip.NewReader(closer)
	if err != nil {
		return gzipr, err
	}

	gw := GzipReader{
		closer: closer,
		reader: gzipr,
	}

	return &gw, nil
}

// Read .
func (gw *GzipReader) Read(p []byte) (int, error) {
	return gw.reader.Read(p)
}

// Close .
func (gw *GzipReader) Close() error {
	defer gw.reader.Close()
	return gw.closer.Close()
}
