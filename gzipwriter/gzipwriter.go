// Copyright (c) 2024 Tiago Melo. All rights reserved.
// Use of this source code is governed by the MIT License that can be found in
// the LICENSE file.

package gzipwriter

import (
	"compress/gzip"
	"io"
)

// Writer is an interface that defines the methods
// required for a writer that supports writing compressed
// data and closing the writer.
type Writer interface {
	// Write writes a compressed byte slice to the underlying writer.
	Write([]byte) (int, error)

	// Close closes the writer, flushing any unwritten data.
	Close() error
}

// New creates a new gzip writer that compresses data
// and writes it to the provided io.Writer.
func New(w io.Writer) Writer {
	return gzip.NewWriter(w)
}
