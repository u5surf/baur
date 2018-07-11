package digest

import (
	"errors"
	"fmt"
	"strings"
)

// Type describes the checksum type
type Type int

const (
	_ Type = iota
	// SHA256 is a sha256 checksum
	SHA256
)

// String returns the textual representation
func (t Type) String() string {
	switch t {
	case SHA256:
		return "sha256"
	default:
		return "undefined"
	}
}

// Digest contains a checksum
type Digest struct {
	Sum  string
	Type Type
}

// String returns '<type>:<checksum>'
func (d *Digest) String() string {
	return fmt.Sprintf("%s:%s", d.Type, d.Sum)
}

// FromString converts a "sha256:<hash> string to Digest
func FromString(in string) (*Digest, error) {
	spl := strings.Split(strings.TrimSpace(in), ":")
	if len(spl) != 2 {
		return nil, errors.New("invalid format, must contain exactly 1 ':'")
	}

	if spl[0] != "sha256" {
		return nil, errors.New("unsupported format %q")
	}

	if len(spl[1]) != 64 {
		return nil, fmt.Errorf("hash length is %d, expected length 64", len(spl[1]))
	}

	return &Digest{
		Sum:  spl[1],
		Type: SHA256,
	}, nil
}