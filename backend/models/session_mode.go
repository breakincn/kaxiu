package models

// SessionMode 表示服务会话的模式（用于客服/叫号模式隔离）。
// 注意：当前阶段仅落地“写入与兼容”，后续会逐步把 status 迁移为 cs_/qs_/qm_/qms_/qmm_ 前缀。
const (
	SessionModeCustomerService = "cs"
	SessionModeQueueAutoSingle = "qs"
	SessionModeQueueAutoMulti  = "qm"
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
	cnt := timeoutCount
	if cnt <= 0 {
		cnt = 1
	}
	return myNo + 2 + 2*(cnt-1)
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
