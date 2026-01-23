package handlers

import (
	"kabao/queue"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type internalQueueTicket struct {
	ID       uint       `json:"id"`
	No       int        `json:"no"`
	CalledAt *time.Time `json:"called_at"`
}

type internalQueueSnapshot struct {
	Tickets []internalQueueTicket `json:"tickets"`
}

func InternalGetOnsiteQueueSnapshot(c *gin.Context) {
	// 如果配置了 token，则要求携带 X-Internal-Token
	if tok := strings.TrimSpace(os.Getenv("KABAO_INTERNAL_TOKEN")); tok != "" {
		got := strings.TrimSpace(c.GetHeader("X-Internal-Token"))
		if got == "" || got != tok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
	}

	merchantIDStr := c.Param("id")
	merchantID64, err := strconv.ParseUint(merchantIDStr, 10, 64)
	if err != nil || merchantID64 == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid merchant id"})
		return
	}
	merchantID := uint(merchantID64)

	date := strings.TrimSpace(c.Query("date"))
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	snap := queue.Default.Snapshot(merchantID, date, queue.QueueTypeOnsite)
	out := internalQueueSnapshot{Tickets: make([]internalQueueTicket, 0, len(snap.Tickets))}
	for _, t := range snap.Tickets {
		out.Tickets = append(out.Tickets, internalQueueTicket{ID: t.ID, No: t.No, CalledAt: t.CalledAt})
	}

	c.JSON(http.StatusOK, gin.H{"data": out})
}
