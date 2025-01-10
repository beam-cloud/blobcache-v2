package blobcache

import (
	"io"

	"github.com/andybalholm/brotli"
	"google.golang.org/grpc/encoding"
)

type BrotliCompressor struct{}

func init() {
	encoding.RegisterCompressor(&BrotliCompressor{})
}

func (c *BrotliCompressor) Name() string {
	return "brotli"
}

func (c *BrotliCompressor) Compress(w io.Writer) (io.WriteCloser, error) {
	return brotli.NewWriter(w), nil
}

func (c *BrotliCompressor) Decompress(r io.Reader) (io.Reader, error) {
	return brotli.NewReader(r), nil
}

func (c *BrotliCompressor) Do(w io.Writer, p []byte) error {
	writer := brotli.NewWriter(w)
	defer writer.Close()
	_, err := writer.Write(p)
	return err
}

func (c *BrotliCompressor) Type() string {
	return "brotli"
}
