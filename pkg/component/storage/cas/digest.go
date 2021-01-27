package cas

import (
	"os"

	"github.com/opencontainers/go-digest"
)

// DigestWriter writer represents a write transaction against the blob store.
type DigestWriter struct {
	fp       *os.File // opened data file
	digester digest.Digester
}

// NewDigestWriter .
func NewDigestWriter(file *os.File) *DigestWriter {
	return &DigestWriter{
		fp:       file,
		digester: digest.Canonical.Digester(),
	}
}

// Digest returns the current digest of the content, up to the current write.
//
// Cannot be called concurrently with `Write`.
func (w *DigestWriter) Digest() digest.Digest {
	return w.digester.Digest()
}

// Write p to the transaction.
//
// Note that writes are unbuffered to the backing file. When writing, it is
// recommended to wrap in a bufio.Writer or, preferably, use io.CopyBuffer.
func (w *DigestWriter) Write(p []byte) (n int, err error) {
	n, err = w.fp.Write(p)
	w.digester.Hash().Write(p[:n])
	return n, err
}

// Close the writer, flushing any unwritten data and leaving the progress in
// tact.
//
// If one needs to resume the transaction, a new writer can be obtained from
// `Ingester.Writer` using the same key. The write can then be continued
// from it was left off.
//
// To abandon a transaction completely, first call close then `IngestManager.Abort` to
// clean up the associated resources.
func (w *DigestWriter) Close() (err error) {
	if w.fp != nil {
		w.fp.Sync()
		err = w.fp.Close()
		w.fp = nil
		return err
	}

	return nil
}
