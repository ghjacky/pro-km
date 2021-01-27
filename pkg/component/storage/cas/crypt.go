package cas

import (
	"crypto/aes"
	"crypto/cipher"
	"io"
)

// EncryptWriter .
type EncryptWriter struct {
	writer io.WriteCloser
	ctr    cipher.Stream
}

// NewEncryptWriter .
func NewEncryptWriter(writer io.WriteCloser, secret string) (io.WriteCloser, error) {
	c, err := aes.NewCipher([]byte(secret))
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	iv := make([]byte, c.BlockSize(), '4')
	ctr := cipher.NewCTR(c, iv)

	return &EncryptWriter{
		writer: writer,
		ctr:    ctr,
	}, nil
}

// Write .
func (e *EncryptWriter) Write(p []byte) (n int, err error) {
	ct := make([]byte, len(p))
	e.ctr.XORKeyStream(ct, p)

	return e.writer.Write(ct)
}

// Close .
func (e *EncryptWriter) Close() error {
	return e.writer.Close()
}

// DecryptReader .
type DecryptReader struct {
	reader io.ReadCloser
	ctr    cipher.Stream
}

// NewDecryptReader .
func NewDecryptReader(reader io.ReadCloser, secret string) (io.ReadCloser, error) {
	c, err := aes.NewCipher([]byte(secret))
	// if there are any errors, handle them
	if err != nil {
		return nil, err
	}

	iv := make([]byte, c.BlockSize(), '4')
	ctr := cipher.NewCTR(c, iv)

	return &DecryptReader{
		reader: reader,
		ctr:    ctr,
	}, nil
}

// Read .
func (dr *DecryptReader) Read(p []byte) (n int, err error) {
	n, err = dr.reader.Read(p)
	dr.ctr.XORKeyStream(p, p)
	return
}

// Close .
func (dr *DecryptReader) Close() error {
	return dr.reader.Close()
}
