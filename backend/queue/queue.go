package queue

import (
	"sync"
	"time"
)

type Store interface {
	Enqueue(merchantID uint, date string, qt QueueType, id uint, startNo int, now time.Time) (Ticket, bool)
	MarkDoneAndCallNext(merchantID uint, date string, qt QueueType, doneID uint, now time.Time) (nextID uint)
	Snapshot(merchantID uint, date string, qt QueueType) Snapshot
}

type QueueType string

const (
	QueueTypeOnsite      QueueType = "onsite"
	QueueTypeAppointment QueueType = "appointment"
)

type Ticket struct {
	ID        uint
	No        int
	EnqueuedAt time.Time
	CalledAt  *time.Time
}

type Snapshot struct {
	Tickets         []Ticket
	ByID            map[uint]Ticket
	CurrentID       uint
	NextID          uint
	CurrentCalledAt *time.Time
}

type dayQueue struct {
	startNo int
	nextNo  int
	order   []uint
	called  map[uint]time.Time
	enqueued map[uint]time.Time
}

type MemStore struct {
	mu sync.Mutex
	// key: merchantID|date|type
	byKey map[string]*dayQueue
}

var Default Store = NewMemStore()

func NewMemStore() *MemStore {
	return &MemStore{byKey: make(map[string]*dayQueue)}
}

func makeKey(merchantID uint, date string, qt QueueType) string {
	return fmtKey(merchantID, date, string(qt))
}

func fmtKey(merchantID uint, date string, qt string) string {
	// avoid fmt to keep deps minimal
	b := make([]byte, 0, 32)
	b = appendUint(b, merchantID)
	b = append(b, '|')
	b = append(b, date...)
	b = append(b, '|')
	b = append(b, qt...)
	return string(b)
}

func appendUint(dst []byte, v uint) []byte {
	// naive uint->decimal
	if v == 0 {
		return append(dst, '0')
	}
	buf := [20]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + byte(v%10))
		v /= 10
	}
	return append(dst, buf[i:]...)
}

func (s *MemStore) Enqueue(merchantID uint, date string, qt QueueType, id uint, startNo int, now time.Time) (Ticket, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := makeKey(merchantID, date, qt)
	dq := s.byKey[k]
	if dq == nil {
		if startNo <= 0 {
			startNo = 1
		}
		dq = &dayQueue{startNo: startNo, nextNo: startNo, order: make([]uint, 0, 16), called: make(map[uint]time.Time), enqueued: make(map[uint]time.Time)}
		s.byKey[k] = dq
	}

	// dedup
	for _, existing := range dq.order {
		if existing == id {
			no := dq.startNo
			for idx, oid := range dq.order {
				if oid == id {
					no = dq.startNo + idx
					break
				}
			}
			enqAt := now
			if t, ok := dq.enqueued[id]; ok {
				enqAt = t
			}
			var calledAt *time.Time
			if t, ok := dq.called[id]; ok {
				tt := t
				calledAt = &tt
			}
			return Ticket{ID: id, No: no, EnqueuedAt: enqAt, CalledAt: calledAt}, false
		}
	}

	dq.order = append(dq.order, id)
	dq.enqueued[id] = now
	no := dq.nextNo
	dq.nextNo++

	// if it's the first in queue, mark called immediately
	if len(dq.order) == 1 {
		dq.called[id] = now
		tt := now
		return Ticket{ID: id, No: no, EnqueuedAt: now, CalledAt: &tt}, true
	}
	return Ticket{ID: id, No: no, EnqueuedAt: now, CalledAt: nil}, true
}

func (s *MemStore) MarkDoneAndCallNext(merchantID uint, date string, qt QueueType, doneID uint, now time.Time) (nextID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := makeKey(merchantID, date, qt)
	dq := s.byKey[k]
	if dq == nil {
		return 0
	}

	// remove doneID from head if it matches
	if len(dq.order) == 0 {
		return 0
	}
	if dq.order[0] == doneID {
		dq.order = dq.order[1:]
	}
	delete(dq.called, doneID)
	delete(dq.enqueued, doneID)

	if len(dq.order) == 0 {
		return 0
	}
	nextID = dq.order[0]
	if _, ok := dq.called[nextID]; !ok {
		dq.called[nextID] = now
	}
	return nextID
}

func (s *MemStore) Snapshot(merchantID uint, date string, qt QueueType) Snapshot {
	s.mu.Lock()
	defer s.mu.Unlock()

	k := makeKey(merchantID, date, qt)
	dq := s.byKey[k]
	out := Snapshot{Tickets: nil, ByID: make(map[uint]Ticket)}
	if dq == nil || len(dq.order) == 0 {
		return out
	}

	out.CurrentID = dq.order[0]
	if len(dq.order) > 1 {
		out.NextID = dq.order[1]
	}

	for idx, id := range dq.order {
		no := dq.startNo + idx
		enqAt := time.Time{}
		if t, ok := dq.enqueued[id]; ok {
			enqAt = t
		}
		var calledAt *time.Time
		if t, ok := dq.called[id]; ok {
			tt := t
			calledAt = &tt
			if id == out.CurrentID {
				out.CurrentCalledAt = calledAt
			}
		}
		t := Ticket{ID: id, No: no, EnqueuedAt: enqAt, CalledAt: calledAt}
		out.Tickets = append(out.Tickets, t)
		out.ByID[id] = t
	}
	return out
}
