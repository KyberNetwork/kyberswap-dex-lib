package crypto

import "github.com/cespare/xxhash/v2"

type Checksumable interface {
	Checksum(salt string) *xxhash.Digest
}

type Checksum struct {
	data Checksumable
	salt string
	hash uint64
}

func NewChecksum(data Checksumable, salt string) *Checksum {
	hash := data.Checksum(salt)
	return &Checksum{
		data: data,
		salt: salt,
		hash: hash.Sum64(),
	}
}

func (c *Checksum) Verify(originalHash uint64) bool {
	return c.hash == originalHash
}

func (c *Checksum) Hash() uint64 {
	return c.hash
}
