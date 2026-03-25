package queue

import (
	"time"
)

type Store interface {
	Enqueue(merchantID uint, date string, qt QueueType, id uint, startNo int, autoCallFirst bool, now time.Time) (Ticket, bool)
	MarkDone(merchantID uint, date string, qt QueueType, doneID uint, now time.Time)
	UnmarkDone(merchantID uint, date string, qt QueueType, id uint)
	GetNo(merchantID uint, date string, qt QueueType, id uint) (no int, ok bool)
	CallNextUncalled(merchantID uint, date string, qt QueueType, now time.Time) (nextID uint)
	Uncall(merchantID uint, date string, qt QueueType, id uint)
	Snapshot(merchantID uint, date string, qt QueueType) Snapshot
}

type QueueType string

const (
	QueueTypeOnsite QueueType = "onsite"
)

type Ticket struct {
	ID         uint
	No         int
	EnqueuedAt time.Time
	CalledAt   *time.Time
}

type Snapshot struct {
	Tickets         []Ticket
	ByID            map[uint]Ticket
	CurrentID       uint
	NextID          uint
	CurrentCalledAt *time.Time
	MaxCalledNo     int
}

var Default Store

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
