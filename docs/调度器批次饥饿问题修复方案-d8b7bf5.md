# 调度器批次饥饿问题修复方案

本方案通过将 `finished` 从主状态推进批次中剥离并独立处理，消除 `ORDER BY id ASC LIMIT 200` 对高 ID 活跃会话的饥饿，确保 `staff_selecting/serving` 等关键状态总能被调度扫描。

## 复核结论

当前实现确实存在你指出的问题：
- `runOnce` 主查询包含 `finished`：@/Users/will/Projects/Go/kabao/backend/scheduler/service_session_scheduler.go#449-454
- 主查询按 `id asc` 且 `limit 200`，当低 ID 的 `finished` 会话很多时，会挤占批次。
- 结果：高 ID 但仍活跃的会话（如 `staff_selecting/start_pending/serving`）可能长期进不了 `advanceOne`。

## 修复目标

1. 主批次只处理“需要推进状态机”的活跃状态。
2. `finished` 技师释放逻辑保留，但改为独立批次执行。
3. 保持行为兼容，不改变既有状态语义。

## 推荐方案（主方案）

### A. 主批次移除 `finished`

把 `runOnce` 的主查询状态集改为：
- `room_selecting`
- `room_locked`
- `staff_selecting`
- `start_pending`
- `delay_pending`
- `serving`
- `auto_finishing`
- `timeout_waiting`
- `timeout_failed`

即从现有查询里删除 `finished`，避免其占用主推进配额。

### B. 新增独立函数处理 `finished` 的技师延迟释放

新增例如：
- `releaseFinishedSessionTechnicians(db, now)`

逻辑：
1. 查询 `status in finished*` 且 `technician_id is not null` 且 `finished_at` 达到延迟释放阈值的记录。
2. 按批次（`LIMIT 200`）循环调用现有 `releaseTechnicianIfNeeded`。
3. 放在 `runOnce` 中单独调用，不与主状态推进共用批次。

### C. runOnce 调用顺序建议

1. `finalizeOverdueManualServingSessions`
2. `finalizeUsagesAfterQueueEnded`
3. `releaseFinishedSessionTechnicians`（新）
4. 主批次活跃状态推进（不含 `finished`）

这样不会影响“完成后延迟释放”能力，同时彻底避免 finished 挤占主推进批次。

## 备选方案（不优先）

1. **改排序为 `updated_at asc`**：可缓解但不能根治（`finished` 仍可能挤占）。
2. **提高 limit**：治标不治本，数据量增长后问题会复现。
3. **按商户分桶轮询**：复杂度高，当前不需要。

## 验收标准

1. 构造 200+ 条低 ID `finished` + 1 条高 ID `staff_selecting`：下一轮调度应能处理高 ID 活跃会话。
2. `finished` 会话仍能按 `auto_idle_after_seconds` 释放技师。
3. 调度总耗时不显著增加（观察日志）。

## 回归测试建议

1. 新增 `runOnce` 批次饥饿测试：确认 `finished` 不再阻塞活跃状态推进。
2. 新增 `releaseFinishedSessionTechnicians` 单测：确认仅满足阈值的 `finished` 会触发释放。
3. 保持现有 `finalizeSession/releaseTechnicianIfNeeded` 测试通过。
