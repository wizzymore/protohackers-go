package reader

import (
	"encoding/binary"
	"fmt"
	"io"
)

// ReadString reads a string from the Reader and stores it in the provided destination.
//
// The function takes an optional byteOrder parameter, which specifies the byte order used to read the string.
// The function returns an error if there is an error reading the string from the Reader.
func ReadString(r io.Reader, dest *string, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	var bb []byte

	var b byte

	binary.Read(r, bo, &b)

	for b != '\x00' {
		bb = append(bb, b)

		if err := binary.Read(r, bo, &b); err != nil {
			return fmt.Errorf("wow.ReadString: %w", err)
		}
	}

	*dest = string(bb)

	return nil
}

// ReadStringFixed reads fixed length string.
func ReadStringFixed(r io.Reader, dest *string, length int, byteOrder ...binary.ByteOrder) error {
	var bo binary.ByteOrder = binary.BigEndian
	if len(byteOrder) > 0 {
		bo = byteOrder[0]
	}

	bb := make([]byte, length)

	if err := binary.Read(r, bo, &bb); err != nil {
		return fmt.Errorf("wow.ReadStringFixed: %w", err)
	}

	*dest = string(bb)

	return nil
}

func ReadByte(r io.Reader) (uint8, error) {
	buf, err := ReadN(r, 1)
	if err != nil {
		return 0, nil
	}

	return buf[0], nil
}

// ReadL reads binary data from the Reader using the LittleEndian encoding and stores it in the provided variable.
//
// v: the variable to store the read data.
// error: returns an error if the reading operation fails.
func ReadL(r io.Reader, v any) error {
	return binary.Read(r, binary.LittleEndian, v)
}

// ReadB reads binary data from the Reader in big endian byte order
//
// v: a variable to store the read data.
// error: an error if the read operation fails.
func ReadB(r io.Reader, v any) error {
	return binary.Read(r, binary.BigEndian, v)
}

// ReadNBytes reads first N bytes from the reader and returns it.
// When can't read from enough bytes from the stream, it will throw an error.
func ReadN(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)

	n2, err := r.Read(buf)
	if err != nil {
		return buf, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		return buf, io.ErrUnexpectedEOF
	}

	return buf, nil
}

// ReadNBytes reads first N bytes from the reader and returns it.
// When can't read from enough bytes from the stream, it will throw an error.
func SkipN(r io.Reader, n int) (int, error) {
	buf := make([]byte, n)

	n2, err := r.Read(buf)
	if err != nil {
		return n2, fmt.Errorf("wow.ReadBytes: %w", err)
	}

	if n2 != n {
		return n2, io.ErrUnexpectedEOF
	}

	return n2, nil
}

// ReadReverseBytes reads and returns n bytes in reverse order from the reader.
//
// It takes an integer n as a parameter, which specifies the number of bytes to read.
// It returns a byte slice containing the read bytes.
func ReadReverseBytes(r io.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)

	err := ReadB(r, buf)
	if err != nil {
		return []byte{}, err
	}

	return ReverseBytes(buf), nil
}

// ReadAll reads all the data from the reader and returns it as a byte slice.
//
// It returns the byte slice containing the data and an error if any.
func ReadAll(r io.Reader) ([]byte, error) {
	return io.ReadAll(r)
}

// ReverseBytes reverses the order of bytes in a byte slice.
//
// data: the byte slice to be reversed.
// []byte: the reversed byte slice.
func ReverseBytes(data []byte) []byte {
	for i, j := 0, len(data)-1; i < j; i, j = i+1, j-1 {
		data[i], data[j] = data[j], data[i]
	}

	return data
}
