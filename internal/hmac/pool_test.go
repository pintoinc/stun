// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package hmac

import (
	"crypto/sha1" //nolint:gosec
	"crypto/sha256"
	"fmt"
	"testing"
)

func BenchmarkHMACSHA1_512(b *testing.B) {
	key := make([]byte, 32)
	buf := make([]byte, 512)
	b.ReportAllocs()
	h := AcquireSHA1(key)
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		h.Write(buf) //nolint:errcheck,gosec
		h.Reset()
		mac := h.Sum(nil)
		buf[0] = mac[0]
	}
}

func BenchmarkHMACSHA1_512_Pool(b *testing.B) {
	key := make([]byte, 32)
	buf := make([]byte, 512)
	tBuf := make([]byte, 0, 512)
	b.ReportAllocs()
	b.SetBytes(int64(len(buf)))
	for i := 0; i < b.N; i++ {
		h := AcquireSHA1(key)
		h.Write(buf) //nolint:errcheck,gosec
		h.Reset()
		mac := h.Sum(tBuf)
		buf[0] = mac[0]
		PutSHA1(h)
	}
}

func TestHMACReset(t *testing.T) {
	for i, tt := range hmacTests() {
		hsh := New(tt.hash, tt.key)
		hsh.(*hmac).resetTo(tt.key) //nolint:forcetypeassert
		if s := hsh.Size(); s != tt.size {
			t.Errorf("Size: got %v, want %v", s, tt.size)
		}
		if b := hsh.BlockSize(); b != tt.blocksize {
			t.Errorf("BlockSize: got %v, want %v", b, tt.blocksize)
		}
		for j := 0; j < 2; j++ {
			n, err := hsh.Write(tt.in)
			if n != len(tt.in) || err != nil {
				t.Errorf("test %d.%d: Write(%d) = %d, %v", i, j, len(tt.in), n, err)

				continue
			}

			// Repetitive Sum() calls should return the same value
			for k := 0; k < 2; k++ {
				sum := fmt.Sprintf("%x", hsh.Sum(nil))
				if sum != tt.out {
					t.Errorf("test %d.%d.%d: have %s want %s", i, j, k, sum, tt.out)
				}
			}

			// Second iteration: make sure reset works.
			hsh.Reset()
		}
	}
}

func TestHMACPool_SHA1(t *testing.T) { //nolint:dupl,cyclop
	for i, tt := range hmacTests() {
		if tt.blocksize != sha1.BlockSize || tt.size != sha1.Size {
			continue
		}
		hsh := AcquireSHA1(tt.key)
		if s := hsh.Size(); s != tt.size {
			t.Errorf("Size: got %v, want %v", s, tt.size)
		}
		if b := hsh.BlockSize(); b != tt.blocksize {
			t.Errorf("BlockSize: got %v, want %v", b, tt.blocksize)
		}
		for j := 0; j < 2; j++ {
			n, err := hsh.Write(tt.in)
			if n != len(tt.in) || err != nil {
				t.Errorf("test %d.%d: Write(%d) = %d, %v", i, j, len(tt.in), n, err)

				continue
			}

			// Repetitive Sum() calls should return the same value
			for k := 0; k < 2; k++ {
				sum := fmt.Sprintf("%x", hsh.Sum(nil))
				if sum != tt.out {
					t.Errorf("test %d.%d.%d: have %s want %s", i, j, k, sum, tt.out)
				}
			}

			// Second iteration: make sure reset works.
			hsh.Reset()
		}
		PutSHA1(hsh)
	}
}

func TestHMACPool_SHA256(t *testing.T) { //nolint:dupl,cyclop
	for i, tt := range hmacTests() {
		if tt.blocksize != sha256.BlockSize || tt.size != sha256.Size {
			continue
		}
		hsh := AcquireSHA256(tt.key)
		if s := hsh.Size(); s != tt.size {
			t.Errorf("Size: got %v, want %v", s, tt.size)
		}
		if b := hsh.BlockSize(); b != tt.blocksize {
			t.Errorf("BlockSize: got %v, want %v", b, tt.blocksize)
		}
		for j := 0; j < 2; j++ {
			n, err := hsh.Write(tt.in)
			if n != len(tt.in) || err != nil {
				t.Errorf("test %d.%d: Write(%d) = %d, %v", i, j, len(tt.in), n, err)

				continue
			}

			// Repetitive Sum() calls should return the same value
			for k := 0; k < 2; k++ {
				sum := fmt.Sprintf("%x", hsh.Sum(nil))
				if sum != tt.out {
					t.Errorf("test %d.%d.%d: have %s want %s", i, j, k, sum, tt.out)
				}
			}

			// Second iteration: make sure reset works.
			hsh.Reset()
		}
		PutSHA256(hsh)
	}
}

func TestAssertBlockSize(t *testing.T) {
	t.Run("Positive", func(*testing.T) {
		h := AcquireSHA1(make([]byte, 0, 1024))
		assertHMACSize(h.(*hmac), sha1.Size, sha1.BlockSize) //nolint:forcetypeassert
	})
	t.Run("Negative", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("should panic")
			}
		}()
		h := AcquireSHA256(make([]byte, 0, 1024))
		assertHMACSize(h.(*hmac), sha1.Size, sha1.BlockSize) //nolint:forcetypeassert
	})
}
