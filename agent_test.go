// SPDX-FileCopyrightText: 2023 The Pion community <https://pion.ly>
// SPDX-License-Identifier: MIT

package stun

import (
	"errors"
	"testing"
	"time"
)

func TestAgent_ProcessInTransaction(t *testing.T) {
	msg := New()
	agent := NewAgent(func(e Event) {
		if e.Error != nil {
			t.Errorf("got error: %s", e.Error)
		}
		if !e.Message.Equal(msg) {
			t.Errorf("%s (got) != %s (expected)", e.Message, msg)
		}
	})
	if err := msg.NewTransactionID(); err != nil {
		t.Fatal(err)
	}
	if err := agent.Start(msg.TransactionID, time.Time{}); err != nil {
		t.Fatal(err)
	}
	if err := agent.Process(msg); err != nil {
		t.Error(err)
	}
	if err := agent.Close(); err != nil {
		t.Error(err)
	}
}

func TestAgent_Process(t *testing.T) {
	msg := New()
	agent := NewAgent(func(e Event) {
		if e.Error != nil {
			t.Errorf("got error: %s", e.Error)
		}
		if !e.Message.Equal(msg) {
			t.Errorf("%s (got) != %s (expected)", e.Message, msg)
		}
	})
	if err := msg.NewTransactionID(); err != nil {
		t.Fatal(err)
	}
	if err := agent.Process(msg); err != nil {
		t.Error(err)
	}
	if err := agent.Close(); err != nil {
		t.Error(err)
	}
	if err := agent.Process(msg); !errors.Is(err, ErrAgentClosed) {
		t.Errorf("closed agent should return <%s>, but got <%s>",
			ErrAgentClosed, err,
		)
	}
}

func TestAgent_Start(t *testing.T) {
	agent := NewAgent(nil)
	id := NewTransactionID()
	deadline := time.Now().AddDate(0, 0, 1)
	if err := agent.Start(id, deadline); err != nil {
		t.Errorf("failed to statt transaction: %s", err)
	}
	if err := agent.Start(id, deadline); !errors.Is(err, ErrTransactionExists) {
		t.Errorf("duplicate start should return <%s>, got <%s>",
			ErrTransactionExists, err,
		)
	}
	if err := agent.Close(); err != nil {
		t.Error(err)
	}
	id = NewTransactionID()
	if err := agent.Start(id, deadline); !errors.Is(err, ErrAgentClosed) {
		t.Errorf("start on closed agent should return <%s>, got <%s>",
			ErrAgentClosed, err,
		)
	}
	if err := agent.SetHandler(nil); !errors.Is(err, ErrAgentClosed) {
		t.Errorf("SetHandler on closed agent should return <%s>, got <%s>",
			ErrAgentClosed, err,
		)
	}
}

func TestAgent_Stop(t *testing.T) {
	called := make(chan Event, 1)
	agent := NewAgent(func(e Event) {
		called <- e
	})
	if err := agent.Stop(transactionID{}); !errors.Is(err, ErrTransactionNotExists) {
		t.Fatalf("unexpected error: %s, should be %s", err, ErrTransactionNotExists)
	}
	id := NewTransactionID()
	timeout := time.Millisecond * 200
	if err := agent.Start(id, time.Now().Add(timeout)); err != nil {
		t.Fatal(err)
	}
	if err := agent.Stop(id); err != nil {
		t.Fatal(err)
	}
	select {
	case e := <-called:
		if !errors.Is(e.Error, ErrTransactionStopped) {
			t.Fatalf("unexpected error: %s, should be %s",
				e.Error, ErrTransactionStopped,
			)
		}
	case <-time.After(timeout * 2):
		t.Fatal("timed out")
	}
	if err := agent.Close(); err != nil {
		t.Fatal(err)
	}
	if err := agent.Close(); !errors.Is(err, ErrAgentClosed) {
		t.Fatalf("a.Close returned %s instead of %s", err, ErrAgentClosed)
	}
	if err := agent.Stop(transactionID{}); !errors.Is(err, ErrAgentClosed) {
		t.Fatalf("unexpected error: %s, should be %s", err, ErrAgentClosed)
	}
}

func TestAgent_GC(t *testing.T) { //nolint:cyclop
	agent := NewAgent(nil)
	shouldTimeOutID := make(map[transactionID]bool)
	deadline := time.Date(2027, time.November, 21,
		23, 0, 0, 0,
		time.UTC,
	)
	gcDeadline := deadline.Add(-time.Second)
	deadlineNotGC := gcDeadline.AddDate(0, 0, -1)
	agent.SetHandler(func(e Event) { //nolint:errcheck,gosec
		id := e.TransactionID
		shouldTimeOut, found := shouldTimeOutID[id]
		if !found {
			t.Error("unexpected transaction ID")
		}
		if shouldTimeOut && !errors.Is(e.Error, ErrTransactionTimeOut) {
			t.Errorf("%x should time out, but got %v", id, e.Error)
		}
		if !shouldTimeOut && errors.Is(e.Error, ErrTransactionTimeOut) {
			t.Errorf("%x should not time out, but got %v", id, e.Error)
		}
	})
	for i := 0; i < 5; i++ {
		id := NewTransactionID()
		shouldTimeOutID[id] = false
		if err := agent.Start(id, deadline); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 5; i++ {
		id := NewTransactionID()
		shouldTimeOutID[id] = true
		if err := agent.Start(id, deadlineNotGC); err != nil {
			t.Fatal(err)
		}
	}
	if err := agent.Collect(gcDeadline); err != nil {
		t.Fatal(err)
	}
	if err := agent.Close(); err != nil {
		t.Error(err)
	}
	if err := agent.Collect(gcDeadline); !errors.Is(err, ErrAgentClosed) {
		t.Errorf("should <%s>, but got <%s>", ErrAgentClosed, err)
	}
}

func BenchmarkAgent_GC(b *testing.B) {
	agent := NewAgent(nil)
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < agentCollectCap; i++ {
		if err := agent.Start(NewTransactionID(), deadline); err != nil {
			b.Fatal(err)
		}
	}
	defer func() {
		if err := agent.Close(); err != nil {
			b.Error(err)
		}
	}()
	b.ReportAllocs()
	gcDeadline := deadline.Add(-time.Second)
	for i := 0; i < b.N; i++ {
		if err := agent.Collect(gcDeadline); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkAgent_Process(b *testing.B) {
	agent := NewAgent(nil)
	deadline := time.Now().AddDate(0, 0, 1)
	for i := 0; i < 1000; i++ {
		if err := agent.Start(NewTransactionID(), deadline); err != nil {
			b.Fatal(err)
		}
	}
	defer func() {
		if err := agent.Close(); err != nil {
			b.Error(err)
		}
	}()
	b.ReportAllocs()
	m := MustBuild(TransactionID)
	for i := 0; i < b.N; i++ {
		if err := agent.Process(m); err != nil {
			b.Fatal(err)
		}
	}
}
