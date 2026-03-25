package scheduler

import (
	"encoding/json"
	"kabao/config"
	"kabao/models"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

const (
	schedulerHeartbeatPersistInterval = 15 * time.Second
	defaultSchedulerStaleMinutes      = 3
)

const (
	schedulerNameServiceSession = "service_session"
	schedulerNameAppointment    = "appointment"
	schedulerNameHandCard       = "hand_card"
)

var (
	schedulerHeartbeatMu            sync.Mutex
	lastSchedulerHeartbeatPersistAt = map[string]time.Time{}
)

type schedulerHeartbeatPayload struct {
	Scheduler string `json:"scheduler"`
	Service   string `json:"service"`
	PID       int    `json:"pid"`
	Hostname  string `json:"hostname"`
	TickAt    string `json:"tick_at"`
}

type SchedulerHealthItem struct {
	Key             string `json:"key"`
	Label           string `json:"label"`
	Status          string `json:"status"`
	LastTickAt      string `json:"last_tick_at,omitempty"`
	LastTickAgeSec  int64  `json:"last_tick_age_sec"`
	ExpectedTickSec int64  `json:"expected_tick_sec"`
	StaleAfterSec   int64  `json:"stale_after_sec"`
	SourceService   string `json:"source_service,omitempty"`
	SourcePID       int    `json:"source_pid,omitempty"`
	SourceHostname  string `json:"source_hostname,omitempty"`
	Message         string `json:"message,omitempty"`
}

type SchedulerHealthReport struct {
	GeneratedAt          string                `json:"generated_at"`
	StaleThresholdMinute int                   `json:"stale_threshold_minutes"`
	OverallStatus        string                `json:"overall_status"`
	OverallMessage       string                `json:"overall_message"`
	Items                []SchedulerHealthItem `json:"items"`
}

func schedulerHeartbeatKey(name string) string {
	return "scheduler_heartbeat_" + strings.TrimSpace(name)
}

func schedulerServiceName() string {
	if v := strings.TrimSpace(config.EnvString("KABAO_SERVICE_NAME")); v != "" {
		return v
	}
	if len(os.Args) > 0 {
		return filepath.Base(os.Args[0])
	}
	return "unknown"
}

func schedulerHeartbeatThrottle(name string, now time.Time) bool {
	schedulerHeartbeatMu.Lock()
	defer schedulerHeartbeatMu.Unlock()
	lastAt := lastSchedulerHeartbeatPersistAt[name]
	if !lastAt.IsZero() && now.Sub(lastAt) < schedulerHeartbeatPersistInterval {
		return true
	}
	lastSchedulerHeartbeatPersistAt[name] = now
	return false
}

func recordSchedulerTick(name string, now time.Time) {
	if config.DB == nil {
		return
	}
	if schedulerHeartbeatThrottle(name, now) {
		return
	}

	host, _ := os.Hostname()
	payload := schedulerHeartbeatPayload{
		Scheduler: name,
		Service:   schedulerServiceName(),
		PID:       os.Getpid(),
		Hostname:  strings.TrimSpace(host),
		TickAt:    now.Format(time.RFC3339Nano),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return
	}

	key := schedulerHeartbeatKey(name)
	var sc models.SystemConfig
	query := config.DB.Where("`key` = ?", key).Limit(1).Find(&sc)
	err = query.Error
	if err == nil && query.RowsAffected > 0 {
		_ = config.DB.Model(&models.SystemConfig{}).Where("id = ?", sc.ID).Update("value", string(raw)).Error
		return
	}
	if err != nil {
		return
	}
	_ = config.DB.Create(&models.SystemConfig{Key: key, Value: string(raw)}).Error
}

func schedulerStaleMinutes(override int) int {
	if override > 0 {
		return override
	}
	if raw := strings.TrimSpace(config.EnvString("KABAO_SCHEDULER_HEALTH_STALE_MINUTES")); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			return n
		}
	}
	return defaultSchedulerStaleMinutes
}

func schedulerDefinitions() []struct {
	Key             string
	Label           string
	ExpectedTickSec int64
} {
	return []struct {
		Key             string
		Label           string
		ExpectedTickSec int64
	}{
		{Key: schedulerNameServiceSession, Label: "服务会话调度器", ExpectedTickSec: int64(schedulerTickInterval / time.Second)},
		{Key: schedulerNameAppointment, Label: "预约调度器", ExpectedTickSec: int64(appointmentSchedulerTickInterval / time.Second)},
		{Key: schedulerNameHandCard, Label: "手牌锁卡调度器", ExpectedTickSec: int64(handCardSchedulerTickInterval / time.Second)},
	}
}

func GetSchedulerHealthReport(db *gorm.DB, now time.Time, staleMinutes int) (SchedulerHealthReport, error) {
	report := SchedulerHealthReport{
		GeneratedAt:          now.Format(time.RFC3339),
		StaleThresholdMinute: schedulerStaleMinutes(staleMinutes),
		OverallStatus:        "healthy",
		OverallMessage:       "调度器运行正常",
		Items:                []SchedulerHealthItem{},
	}
	if db == nil {
		report.OverallStatus = "unknown"
		report.OverallMessage = "数据库未初始化，无法判断调度器状态"
		return report, nil
	}

	staleAfterSec := int64(report.StaleThresholdMinute * 60)
	overallStatus := "healthy"
	overallMessage := "调度器运行正常"

	for _, def := range schedulerDefinitions() {
		item := SchedulerHealthItem{
			Key:             def.Key,
			Label:           def.Label,
			Status:          "missing",
			ExpectedTickSec: def.ExpectedTickSec,
			StaleAfterSec:   staleAfterSec,
			Message:         "未检测到心跳",
		}
		var sc models.SystemConfig
		query := db.Where("`key` = ?", schedulerHeartbeatKey(def.Key)).Limit(1).Find(&sc)
		if err := query.Error; err != nil || query.RowsAffected == 0 {
			if err != nil {
				item.Status = "error"
				item.Message = err.Error()
				overallStatus = "degraded"
				overallMessage = "存在调度器状态读取异常"
			} else if overallStatus == "healthy" {
				overallStatus = "degraded"
				overallMessage = "存在调度器尚未上报心跳"
			}
			report.Items = append(report.Items, item)
			continue
		}

		var payload schedulerHeartbeatPayload
		if err := json.Unmarshal([]byte(sc.Value), &payload); err != nil {
			item.Status = "error"
			item.Message = "心跳数据格式错误"
			overallStatus = "degraded"
			overallMessage = "存在调度器心跳数据异常"
			report.Items = append(report.Items, item)
			continue
		}
		item.SourceService = payload.Service
		item.SourcePID = payload.PID
		item.SourceHostname = payload.Hostname
		item.LastTickAt = payload.TickAt

		tickAt, err := time.Parse(time.RFC3339Nano, payload.TickAt)
		if err != nil {
			item.Status = "error"
			item.Message = "心跳时间格式错误"
			overallStatus = "degraded"
			overallMessage = "存在调度器心跳数据异常"
			report.Items = append(report.Items, item)
			continue
		}

		ageSec := int64(now.Sub(tickAt).Seconds())
		if ageSec < 0 {
			ageSec = 0
		}
		item.LastTickAgeSec = ageSec
		if ageSec > staleAfterSec {
			item.Status = "stale"
			item.Message = "超过阈值时间未检测到新 tick"
			overallStatus = "degraded"
			overallMessage = "存在调度器超过阈值时间未 tick"
		} else {
			item.Status = "healthy"
			item.Message = "最近 tick 正常"
		}
		report.Items = append(report.Items, item)
	}

	report.OverallStatus = overallStatus
	report.OverallMessage = overallMessage
	return report, nil
}
