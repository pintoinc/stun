// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

//go:build !js
// +build !js

package stun

import (
	"errors"
	"strings"
	"testing"
)

func TestSoftware_GetFrom(t *testing.T) {
	msg := New()
	val := "Client v0.0.1"
	msg.Add(AttrSoftware, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	software := new(Software)
	if _, err := m2.ReadFrom(msg.reader()); err != nil {
		t.Error(err)
	}
	if err := software.GetFrom(msg); err != nil {
		t.Fatal(err)
	}
	if software.String() != val {
		t.Errorf("Expected %q, got %q.", val, software)
	}

	sAttr, ok := msg.Attributes.Get(AttrSoftware)
	if !ok {
		t.Error("software attribute should be found")
	}
	s := sAttr.String()
	if !strings.HasPrefix(s, "SOFTWARE:") {
		t.Error("bad string representation", s)
	}
}

func TestSoftware_AddTo_Invalid(t *testing.T) {
	m := New()
	s := make(Software, 1024)
	if err := s.AddTo(m); !IsAttrSizeOverflow(err) {
		t.Errorf("AddTo should return *AttrOverflowErr, got: %v", err)
	}
	if err := s.GetFrom(m); !errors.Is(err, ErrAttributeNotFound) {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}

func TestSoftware_AddTo_Regression(t *testing.T) {
	// s.AddTo checked len(m.Raw) instead of len(s.Raw).
	m := &Message{Raw: make([]byte, 2048)}
	s := make(Software, 100)
	if err := s.AddTo(m); err != nil {
		t.Errorf("AddTo should return <nil>, got: %v", err)
	}
}

func BenchmarkUsername_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	u := Username("test")
	for i := 0; i < b.N; i++ {
		if err := u.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func BenchmarkUsername_GetFrom(b *testing.B) {
	b.ReportAllocs()
	m := new(Message)
	Username("test").AddTo(m) //nolint:errcheck,gosec
	var u Username
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := u.GetFrom(m); err != nil {
			b.Fatal(err)
		}
		u = u[:0]
	}
}

func TestUsername(t *testing.T) {
	username := "username"
	uName := NewUsername(username)
	msg := new(Message)
	msg.WriteHeader()
	t.Run("Bad length", func(t *testing.T) {
		badU := make(Username, 600)
		if err := badU.AddTo(msg); !IsAttrSizeOverflow(err) {
			t.Errorf("AddTo should return *AttrOverflowErr, got: %v", err)
		}
	})
	t.Run("AddTo", func(t *testing.T) {
		if err := uName.AddTo(msg); err != nil {
			t.Error("errored:", err)
		}
		t.Run("GetFrom", func(t *testing.T) {
			got := new(Username)
			if err := got.GetFrom(msg); err != nil {
				t.Error("errored:", err)
			}
			if got.String() != username {
				t.Errorf("expedted: %s, got: %s", username, got)
			}
			t.Run("Not found", func(t *testing.T) {
				m := new(Message)
				u := new(Username)
				if err := u.GetFrom(m); !errors.Is(err, ErrAttributeNotFound) {
					t.Error("Should error")
				}
			})
		})
	})
	t.Run("No allocations", func(t *testing.T) {
		m := new(Message)
		m.WriteHeader()
		u := NewUsername("username")
		if allocs := testing.AllocsPerRun(10, func() {
			if err := u.AddTo(m); err != nil {
				t.Error(err)
			}
			m.Reset()
		}); allocs > 0 {
			t.Errorf("got %f allocations, zero expected", allocs)
		}
	})
}

func TestRealm_GetFrom(t *testing.T) {
	msg := New()
	val := "realm"
	msg.Add(AttrRealm, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	r := new(Realm)
	if err := r.GetFrom(m2); !errors.Is(err, ErrAttributeNotFound) {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
	if _, err := m2.ReadFrom(msg.reader()); err != nil {
		t.Error(err)
	}
	if err := r.GetFrom(msg); err != nil {
		t.Fatal(err)
	}
	if r.String() != val {
		t.Errorf("Expected %q, got %q.", val, r)
	}

	rAttr, ok := msg.Attributes.Get(AttrRealm)
	if !ok {
		t.Error("realm attribute should be found")
	}
	s := rAttr.String()
	if !strings.HasPrefix(s, "REALM:") {
		t.Error("bad string representation", s)
	}
}

func TestRealm_AddTo_Invalid(t *testing.T) {
	m := New()
	r := make(Realm, 1024)
	if err := r.AddTo(m); !IsAttrSizeOverflow(err) {
		t.Errorf("AddTo should return *AttrOverflowErr, got: %v", err)
	}
	if err := r.GetFrom(m); !errors.Is(err, ErrAttributeNotFound) {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}

func TestNonce_GetFrom(t *testing.T) {
	msg := New()
	val := "example.org"
	msg.Add(AttrNonce, []byte(val))
	msg.WriteHeader()

	m2 := &Message{
		Raw: make([]byte, 0, 256),
	}
	var nonce Nonce
	if _, err := m2.ReadFrom(msg.reader()); err != nil {
		t.Error(err)
	}
	if err := nonce.GetFrom(msg); err != nil {
		t.Fatal(err)
	}
	if nonce.String() != val {
		t.Errorf("Expected %q, got %q.", val, nonce)
	}

	nAttr, ok := msg.Attributes.Get(AttrNonce)
	if !ok {
		t.Error("nonce attribute should be found")
	}
	s := nAttr.String()
	if !strings.HasPrefix(s, "NONCE:") {
		t.Error("bad string representation", s)
	}
}

func TestNonce_AddTo_Invalid(t *testing.T) {
	m := New()
	n := make(Nonce, 1024)
	if err := n.AddTo(m); !IsAttrSizeOverflow(err) {
		t.Errorf("AddTo should return *AttrOverflowErr, got: %v", err)
	}
	if err := n.GetFrom(m); !errors.Is(err, ErrAttributeNotFound) {
		t.Errorf("GetFrom should return %q, got: %v", ErrAttributeNotFound, err)
	}
}

func TestNonce_AddTo(t *testing.T) {
	m := New()
	n := Nonce("example.org")
	if err := n.AddTo(m); err != nil {
		t.Error(err)
	}
	v, err := m.Get(AttrNonce)
	if err != nil {
		t.Error(err)
	}
	if string(v) != "example.org" {
		t.Errorf("bad nonce %q", v)
	}
}

func BenchmarkNonce_AddTo(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := NewNonce("nonce")
	for i := 0; i < b.N; i++ {
		if err := n.AddTo(m); err != nil {
			b.Fatal(err)
		}
		m.Reset()
	}
}

func BenchmarkNonce_AddTo_BadLength(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := make(Nonce, 2048)
	for i := 0; i < b.N; i++ {
		if err := n.AddTo(m); err == nil {
			b.Fatal("should error")
		}
		m.Reset()
	}
}

func BenchmarkNonce_GetFrom(b *testing.B) {
	b.ReportAllocs()
	m := New()
	n := NewNonce("nonce")
	n.AddTo(m) //nolint:errcheck,gosec
	for i := 0; i < b.N; i++ {
		n.GetFrom(m) //nolint:errcheck,gosec
	}
}
