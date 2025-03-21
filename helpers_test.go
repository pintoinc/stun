// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"fmt"
	"testing"

	"github.com/pion/stun/v3/internal/testutil"
)

func BenchmarkBuildOverhead(b *testing.B) {
	var (
		msgType  = BindingRequest
		username = NewUsername("username")
		nonce    = NewNonce("nonce")
		realm    = NewRealm("example.org")
	)
	b.Run("Build", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(&msgType, &username, &nonce, &realm, &Fingerprint) //nolint:errcheck,gosec
		}
	})
	b.Run("BuildNonPointer", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Build(msgType, username, nonce, realm, Fingerprint) //nolint:errcheck,gosec //nolint:errcheck,gosec
		}
	})
	b.Run("Raw", func(b *testing.B) {
		b.ReportAllocs()
		m := new(Message)
		for i := 0; i < b.N; i++ {
			m.Reset()
			m.WriteHeader()
			m.SetType(msgType)
			username.AddTo(m)    //nolint:errcheck,gosec
			nonce.AddTo(m)       //nolint:errcheck,gosec
			realm.AddTo(m)       //nolint:errcheck,gosec
			Fingerprint.AddTo(m) //nolint:errcheck,gosec
		}
	})
}

func TestMessage_Apply(t *testing.T) {
	var (
		integrity = NewShortTermIntegrity("password")
		decoded   = new(Message)
	)
	msg, err := Build(BindingRequest, TransactionID,
		NewUsername("username"),
		NewNonce("nonce"),
		NewRealm("example.org"),
		integrity,
		Fingerprint,
	)
	if err != nil {
		t.Fatal("failed to build:", err)
	}
	if err = msg.Check(Fingerprint, integrity); err != nil {
		t.Fatal(err)
	}
	if _, err := decoded.Write(msg.Raw); err != nil {
		t.Fatal(err)
	}
	if !decoded.Equal(msg) {
		t.Error("not equal")
	}
	if err := integrity.Check(decoded); err != nil {
		t.Fatal(err)
	}
}

type errReturner struct {
	Err error
}

var errTError = errors.New("tError")

func (e errReturner) AddTo(*Message) error {
	return e.Err
}

func (e errReturner) Check(*Message) error {
	return e.Err
}

func (e errReturner) GetFrom(*Message) error {
	return e.Err
}

func TestHelpersErrorHandling(t *testing.T) {
	m := New()
	errReturn := errReturner{Err: errTError}
	if err := m.Build(errReturn); !errors.Is(err, errReturn.Err) {
		t.Error(err, "!=", errReturn.Err)
	}
	if err := m.Check(errReturn); !errors.Is(err, errReturn.Err) {
		t.Error(err, "!=", errReturn.Err)
	}
	if err := m.Parse(errReturn); !errors.Is(err, errReturn.Err) {
		t.Error(err, "!=", errReturn.Err)
	}
	t.Run("MustBuild", func(t *testing.T) {
		t.Run("Positive", func(*testing.T) {
			MustBuild(NewTransactionIDSetter(transactionID{}))
		})
		defer func() {
			if p, ok := recover().(error); !ok || !errors.Is(p, errReturn.Err) {
				t.Errorf("%s != %s",
					p, errReturn.Err,
				)
			}
		}()
		MustBuild(errReturn)
	})
}

func TestMessage_ForEach(t *testing.T) { //nolint:cyclop
	initial := New()
	if err := initial.Build(
		NewRealm("realm1"), NewRealm("realm2"),
	); err != nil {
		t.Fatal(err)
	}
	newMessage := func() *Message {
		m := New()
		if err := m.Build(
			NewRealm("realm1"), NewRealm("realm2"),
		); err != nil {
			t.Fatal(err)
		}

		return m
	}
	t.Run("NoResults", func(t *testing.T) {
		m := newMessage()
		if !m.Equal(initial) {
			t.Error("m should be equal to initial")
		}
		if err := m.ForEach(AttrUsername, func(*Message) error {
			t.Error("should not be called")

			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if !m.Equal(initial) {
			t.Error("m should be equal to initial")
		}
	})
	t.Run("ReturnOnError", func(t *testing.T) {
		m := newMessage()
		var calls int
		if err := m.ForEach(AttrRealm, func(*Message) error {
			if calls > 0 {
				t.Error("called multiple times")
			}
			calls++

			return ErrAttributeNotFound
		}); !errors.Is(err, ErrAttributeNotFound) {
			t.Fatal(err)
		}
		if !m.Equal(initial) {
			t.Error("m should be equal to initial")
		}
	})
	t.Run("Positive", func(t *testing.T) {
		msg := newMessage()
		var realms []string
		if err := msg.ForEach(AttrRealm, func(m *Message) error {
			var realm Realm
			if err := realm.GetFrom(m); err != nil {
				return err
			}
			realms = append(realms, realm.String())

			return nil
		}); err != nil {
			t.Fatal(err)
		}
		if len(realms) != 2 {
			t.Fatal("expected 2 realms")
		}
		if realms[0] != "realm1" {
			t.Error("bad value for 1 realm")
		}
		if realms[1] != "realm2" {
			t.Error("bad value for 2 realm")
		}
		if !msg.Equal(initial) {
			t.Error("m should be equal to initial")
		}
		t.Run("ZeroAlloc", func(t *testing.T) {
			msg = newMessage()
			var realm Realm
			testutil.ShouldNotAllocate(t, func() {
				if err := msg.ForEach(AttrRealm, realm.GetFrom); err != nil {
					t.Fatal(err)
				}
			})
			if !msg.Equal(initial) {
				t.Error("m should be equal to initial")
			}
		})
	})
}

func ExampleMessage_ForEach() {
	m := MustBuild(NewRealm("realm1"), NewRealm("realm2"))
	if err := m.ForEach(AttrRealm, func(m *Message) error {
		var r Realm
		if err := r.GetFrom(m); err != nil {
			return err
		}
		fmt.Println(r)

		return nil
	}); err != nil {
		fmt.Println("error:", err)
	}

	// Output:
	// realm1
	// realm2
}

func BenchmarkMessage_ForEach(b *testing.B) {
	b.ReportAllocs()
	m := MustBuild(
		NewRealm("realm1"),
		NewRealm("realm2"),
		NewRealm("realm3"),
		NewRealm("realm4"),
	)
	for i := 0; i < b.N; i++ {
		if err := m.ForEach(AttrRealm, func(*Message) error {
			return nil
		}); err != nil {
			b.Fatal(err)
		}
	}
}
