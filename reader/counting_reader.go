package reader

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/rs/zerolog/log"
)

type CountingReader struct {
	reader    io.Reader
	BytesRead int
}

// Read reads data into the provided byte slice.
// It returns the number of bytes read and any error encountered.
func (cr *CountingReader) Read(b []byte) (int, error) {
	readed, err := cr.reader.Read(b)
	cr.BytesRead += readed

	return readed, err
}

// ReadString reads a string from the Reader and stores it in the provided destination.
//
// The function takes an optional byteOrder parameter, which specifies the byte order used to read the string.
// The function returns an error if there is an error reading the string from the Reader.
func (cr *CountingReader) ReadString(dest *string, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	var bb []byte

	var b byte

	binary.Read(cr, bo, &b)

	for b != '\x00' {
		bb = append(bb, b)

		if err := binary.Read(cr, binary.LittleEndian, &b); err != nil {
			return fmt.Errorf("wow.ReadString: %w", err)
		}
	}

	*dest = string(bb)

	return nil
}

// ReadStringFixed reads fixed length string.
func (cr *CountingReader) ReadStringFixed(dest *string, length int, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.LittleEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	bb := make([]byte, length)

	if err := binary.Read(cr, bo, &bb); err != nil {
		return fmt.Errorf("wow.ReadStringFixed: %w", err)
	}

	*dest = string(bb)

	return nil
}

// ReadBytes reads data into p.
// It returns the number of bytes read and any error encountered.
func (cr *CountingReader) ReadBytes(p []byte) (int, error) {
	return cr.Read(p)
}

// ReadL reads binary data from the Reader using the LittleEndian encoding and stores it in the provided variable.
//
// v: the variable to store the read data.
// error: returns an error if the reading operation fails.
func (cr *CountingReader) ReadL(v any) error {
	return binary.Read(cr, binary.LittleEndian, v)
}

// ReadB reads binary data from the Reader in big endian byte order
//
// v: a variable to store the read data.
// error: an error if the read operation fails.
func (cr *CountingReader) ReadB(v any) error {
	return binary.Read(cr, binary.BigEndian, v)
}

// ReadNBytes reads first N bytes from the reader and returns it.
// When can't read from enough bytes from the stream, it will throw an error.
func (cr *CountingReader) ReadN(n int) ([]byte, error) {
	buf := make([]byte, n)

	n2, err := cr.ReadBytes(buf)
	if err != nil {
		return buf, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		log.Warn().Err(err).Msgf("readed %d instead of required: %d", n2, n)

		return buf, io.ErrUnexpectedEOF
	}

	return buf, nil
}

// ReadNBytes reads first N bytes from the reader and returns it.
// When can't read from enough bytes from the stream, it will throw an error.
func (cr *CountingReader) SkipN(n int) (int, error) {
	buf := make([]byte, n)

	n2, err := cr.ReadBytes(buf)
	if err != nil {
		return n2, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		log.Warn().Err(err).Msgf("skipped %d instead of required: %d", n2, n)

		return n2, io.ErrUnexpectedEOF
	}

	return n2, nil
}

// ReadReverseBytes reads and returns n bytes in reverse order from the reader.
//
// It takes an integer n as a parameter, which specifies the number of bytes to read.
// It returns a byte slice containing the read bytes.
func (cr *CountingReader) ReadReverseBytes(n int) []byte {
	buf := make([]byte, n)

	err := cr.ReadB(buf)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

	return ReverseBytes(buf)
}

// ReadAll reads all the data from the reader and returns it as a byte slice.
//
// It returns the byte slice containing the data and an error if any.
func (cr *CountingReader) ReadAll() ([]byte, error) {
	return io.ReadAll(cr)
}
