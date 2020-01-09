package typedbucket

import (
	"bytes"
	"errors"

	"github.com/ledgerwatch/bolt"
	"github.com/ledgerwatch/turbo-geth/ethdb/codecpool"
)

type Uint64 struct {
	*bolt.Bucket
}

func NewUint64(b *bolt.Bucket) *Uint64 {
	return &Uint64{b}
}

func (b *Uint64) Get(key []byte) (uint64, bool) {
	value, _ := b.Bucket.Get(key)
	if value == nil {
		return 0, false
	}
	var v uint64
	decoder := codecpool.Decoder(bytes.NewReader(value))
	defer codecpool.Return(decoder)

	decoder.MustDecode(&v)
	return v, true
}

func (b *Uint64) Increment(key []byte) error {
	v, _ := b.Get(key)
	return b.Put(key, v+1)
}

func (b *Uint64) Decrement(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		// return ethdb.ErrNotFound
		return errors.New("not found key")
	}
	if v == 0 {
		return errors.New("could not decrement zero")
	}
	return b.Put(key, v-1)
}

func (b *Uint64) DecrementIfExist(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		return nil
	}
	if v == 0 {
		return errors.New("could not decrement zero")
	}

	return b.Put(key, v-1)
}

func (b *Uint64) Put(key []byte, value uint64) error {
	var buf bytes.Buffer

	encoder := codecpool.Encoder(&buf)
	defer codecpool.Return(encoder)

	encoder.MustEncode(&value)
	return b.Bucket.Put(key, buf.Bytes())
}

func (b *Uint64) ForEach(fn func([]byte, uint64) error) error {
	return b.Bucket.ForEach(func(k, v []byte) error {
		var value uint64
		decoder := codecpool.Decoder(bytes.NewReader(v))
		defer codecpool.Return(decoder)

		decoder.MustDecode(&value)
		return fn(k, value)
	})
}

type Int struct {
	*bolt.Bucket
}

func NewInt(b *bolt.Bucket) *Int {
	return &Int{b}
}

func (b *Int) Get(key []byte) (int, bool) {
	value, _ := b.Bucket.Get(key)
	if value == nil {
		return 0, false
	}
	var v int
	decoder := codecpool.Decoder(bytes.NewReader(value))
	defer codecpool.Return(decoder)

	decoder.MustDecode(&v)
	return v, true
}

func (b *Int) Put(key []byte, value int) error {
	var buf bytes.Buffer

	encoder := codecpool.Encoder(&buf)
	defer codecpool.Return(encoder)

	encoder.MustEncode(&value)
	return b.Bucket.Put(key, buf.Bytes())
}

func (b *Int) Increment(key []byte) error {
	v, _ := b.Get(key)
	return b.Put(key, v+1)
}

func (b *Int) Decrement(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		// return ethdb.ErrNotFound
		return errors.New("not found key")
	}
	return b.Put(key, v-1)
}

func (b *Int) DecrementIfExist(key []byte) error {
	v, ok := b.Get(key)
	if !ok {
		return nil
	}
	return b.Put(key, v-1)
}

func (b *Int) ForEach(fn func([]byte, int) error) error {
	return b.Bucket.ForEach(func(k, v []byte) error {
		var value int
		decoder := codecpool.Decoder(bytes.NewReader(v))
		defer codecpool.Return(decoder)

		decoder.MustDecode(&value)
		return fn(k, value)
	})
}
