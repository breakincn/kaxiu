package models

import (
	"errors"
	"strings"
)

// SessionMode 表示服务会话的模式（用于客服/叫号模式隔离）。
// 注意：当前阶段仅落地“写入与兼容”，后续会逐步把 status 迁移为 cs_/qs_/qm_/qms_/qmm_ 前缀。
const (
	SessionModeCustomerService   = "cs"
	SessionModeQueueAutoSingle   = "qs"
	SessionModeQueueAutoMulti    = "qm"
	SessionModeQueueManualSingle = "qms"
	SessionModeQueueManualMulti  = "qmm"
	SessionModeOrderComplete     = "oc"
	SessionModeSimple            = "simple"
)

func ResolveSessionMode(m *Merchant) string {
	if m == nil {
		return ""
	}
	if m.SupportCustomerServiceMode {
		return SessionModeCustomerService
	}
	if m.SupportQueue {
		if m.QueueMode == "auto" {
			if m.SupportMultiCustomerService {
				return SessionModeQueueAutoMulti
			}
			return SessionModeQueueAutoSingle
		}
		if m.QueueMode == "manual" {
			if m.SupportMultiCustomerService {
				return SessionModeQueueManualMulti
			}
			return SessionModeQueueManualSingle
		}
	}
	if m.SupportOrderComplete {
		return SessionModeOrderComplete
	}
	return SessionModeSimple
}

func QsTimeoutWindowEndNo(myNo int, timeoutCount int) int {
	if myNo <= 0 {
		return 0
	}
	_ = timeoutCount
	return myNo + 3
}

func QsTimeoutWaitingExpired(currentNo int, myNo int, timeoutCount int) bool {
	if currentNo <= 0 || myNo <= 0 {
		return false
	}
	end := QsTimeoutWindowEndNo(myNo, timeoutCount)
	if end <= 0 {
		return false
	}
	return currentNo >= end+1
}

// NormalizeLegacySessionMode 返回会话的有效模式。
// 当 session_mode 为空（模式追踪引入前的历史数据）时，回退到根据商户配置推断的模式。
// 所有需要判断"实际模式"的调用方应优先使用此函数，而非直接检查 s.SessionMode == ""。
func NormalizeLegacySessionMode(s *ServiceSession, m *Merchant) string {
	if s == nil {
		return ""
	}
	if s.SessionMode != "" {
		return s.SessionMode
	}
	if m == nil {
		return ""
	}
	return ResolveSessionMode(m)
}

// IsQueueMode 报告 mode 是否属于叫号族（qs/qm/qms/qmm）。
func IsQueueMode(mode string) bool {
	switch mode {
	case SessionModeQueueAutoSingle, SessionModeQueueAutoMulti,
		SessionModeQueueManualSingle, SessionModeQueueManualMulti:
		return true
	}
	return false
}

// IsCSMode 报告 mode 是否为客服模式（cs）。
func IsCSMode(mode string) bool {
	return mode == SessionModeCustomerService
}

// ErrSessionModeMismatch 由 ValidateSessionModeForEntry 在会话模式与入口期望模式不兼容时返回。
var ErrSessionModeMismatch = errors.New("会话模式与当前流程不匹配，请刷新后重试")

// ValidateSessionModeForEntry 检查会话的有效模式是否与商户配置所暗示的入口模式兼容。
//
// 规则：
//   - CS 入口（m.SupportCustomerServiceMode=true）→ 会话必须为 cs 或历史空 mode。
//   - Queue 入口（m.SupportQueue=true, !SupportCustomerServiceMode）→ 会话必须为叫号族 mode 或历史空 mode。
//   - 历史空 session_mode 始终视为兼容（避免历史数据误拦截）。
func ValidateSessionModeForEntry(s *ServiceSession, m *Merchant) error {
	if s == nil || m == nil {
		return nil
	}
	sMode := s.SessionMode
	if sMode == "" {
		return nil // 历史数据，始终兼容
	}
	expected := ResolveSessionMode(m)
	if expected == "" {
		return nil
	}
	if IsCSMode(expected) && !IsCSMode(sMode) {
		return ErrSessionModeMismatch
	}
	if IsQueueMode(expected) && !IsQueueMode(sMode) {
		return ErrSessionModeMismatch
	}
	return nil
}

// IsModeConsistentWithStatus 检查会话的 session_mode 字段是否与 status 中嵌入的前缀一致。
// 用于调度器在无需加载商户信息的情况下检测跨模式污染。
// 历史空 session_mode 始终视为一致。
func IsModeConsistentWithStatus(s *ServiceSession) bool {
	if s == nil || s.SessionMode == "" {
		return true
	}
	st := strings.TrimSpace(s.Status)
	for _, p := range knownStatusPrefixes {
		if strings.HasPrefix(st, p) {
			return p == s.SessionMode+"_"
		}
	}
	// 无前缀状态（历史裸状态）视为一致
	return true
}
