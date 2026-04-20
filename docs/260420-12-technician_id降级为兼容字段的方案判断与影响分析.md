# 问答记录
**时间**：2026-04-20 12:05:00 +0800
**提问**：方案判断 判断：是否应该把 technician_id 从“展示主负责人”降级为纯兼容字段 落地影响分析 直接列出：如果取消“伪主服务人员”展示，前后端需要改哪些点，风险有多大
**回答**：结论：**应该**把 `technician_id` 从“展示主负责人”降级为**纯兼容字段**，但建议采用“**先取消展示语义，再逐步缩小业务依赖**”的方式落地，而不是一次性完全删除。

原因很明确：

1. 当前真实业务语义已经是多人模型
   - `service_technician_ids` 表示绑定服务人员集合
   - `start_confirmed_technician_ids` 表示已扫码确认人员集合
   - 这两个字段已经足以表达业务事实

2. `technician_id` 现在主要是历史兼容字段
   - 某些旧查询、旧预加载、旧前端展示仍在用
   - 但它并不能稳定表达“真实先确认的人”或“当前唯一负责人”

3. 继续把它当“主负责人”展示会制造歧义
   - 在 `[A, B]` 且 `B` 先核销/先确认的场景里
   - `technician_id` 仍可能是 `A`
   - 真实确认人却是 `B`
   - 页面若显示 `A`，就会与真实业务动作冲突

所以最合理的策略是：

- **业务解释层面**：把 `technician_id` 降级为兼容字段
- **展示层面**：不再把 `technician_id` 当“主负责人”直接展示
- **数据兼容层面**：暂时保留该字段，供旧逻辑、索引、关联和兼容接口继续使用

---

## 一、是否应该降级

### 结论

**应该降级。**

### 不建议继续保留“主负责人展示语义”的原因

1. 没有真实业务需求支撑
   - 你已经明确说明：当前没有主次服务人员区分需求
   - 那么界面上突出一个“主负责人”本身就是伪语义

2. 当前数据模型已经能更准确表达业务
   - 谁参与：`service_technician_ids`
   - 谁已确认：`start_confirmed_technician_ids`
   - 最近是谁操作：`last_technician_id`

3. `technician_id` 与真实动作可能不一致
   - 默认常取服务人员列表第一个
   - 这只是实现习惯，不是业务事实

4. 保留展示会持续误导前端和运营
   - 页面展示 A
   - 实际确认人是 B
   - 后续排查问题、解释状态、看板统计都会混乱

### 不建议一步删掉 `technician_id` 的原因

1. 后端还有不少逻辑直接依赖它
2. 前端仍有组件优先展示 `session.technician`
3. 列表、自筛选、超时释放、在服判断等逻辑仍把它作为兼容锚点
4. 一次性删除会把“展示问题”扩大成“状态机/统计/接口兼容问题”

所以建议：

- **短期**：降级展示语义，保留兼容字段
- **中期**：逐步把后端业务判断改到多人字段上
- **长期**：评估是否彻底去除 `technician_id` 的强依赖

---

## 二、如果取消“伪主服务人员”展示，前端要改哪些点

### 1. 服务单卡片展示

当前组件：`frontend/src/components/ServiceSessionItem.vue`

现状：
- `effectiveTechnician` 优先取 `session.technician`
- 其次才 `last_technician`
- 这会把 `technician_id` 对应的人显示成“技师”

建议：
- 不再展示单个“技师：xxx”
- 改为展示：
  - `服务人员：A、B`
  - 对已确认人员加状态，例如 `A（已扫码）`、`B（未扫码）`
- 若只想最小改动：
  - 优先显示 `service_technicians`
  - 不再优先 `technician`

风险：**低**
- 纯展示调整
- 只要接口已返回 `service_technicians` 即可

### 2. Dashboard 技师相关会话归属

当前文件：`frontend/src/views/merchant/Dashboard.vue`

现状：
- `getSessionServiceTechnicianIDs` 已经综合使用：
  - `service_technicians`
  - `service_technician_ids`
  - `start_confirmed_technician_ids`
  - `technician_id`
- 归属判断总体已偏向多人模型

建议：
- 归属判断保留兼容兜底，但优先级改为：
  1. `service_technicians`
  2. `service_technician_ids`
  3. `start_confirmed_technician_ids`
  4. 最后才 `technician_id`
- 对“已确认”展示只基于：
  - `start_confirmed_technician_ids`
  - 或 `service_technicians[*].service_start_confirmed`

风险：**低到中**
- 逻辑已有基础
- 主要是清理兼容优先级

### 3. Table / ServiceSessions 等页面的“我的服务中”过滤

当前可见问题：
- 有些过滤直接写死 `session.technician_id === techId`
- 例如：
  - `frontend/src/views/merchant/Table.vue`
  - `frontend/src/views/merchant/ServiceSessions.vue`

建议：
- 改成统一使用“多人归属判断函数”
- 不要再直接用 `technician_id === 当前技师`
- 对“服务中/待开始”列表，应基于：
  - `service_technician_ids`
  - `start_confirmed_technician_ids`
  - 必要时再兼容 `technician_id`

风险：**中**
- 会影响技师端看到哪些单
- 需要人工回归验证多个状态：`start_pending` / `delay_pending` / `serving`

### 4. 前端展示文案

建议：
- 把“技师”单数展示调整成“服务人员”
- 对已确认增加明确标签，而不是默认突出某一人

风险：**低**
- 主要是文案和展示方式调整

---

## 三、如果取消“伪主服务人员”展示，后端要改哪些点

### 1. 保留 `technician_id`，但明确只做兼容字段

当前建议：
- 后端模型不删字段
- 接口仍可返回 `technician`，但前端不再把它当主展示
- 对外语义上弱化它

风险：**低**
- 不改数据库结构
- 不影响旧数据兼容

### 2. 服务会话详情/列表接口的展示重心

当前接口：
- `GetServiceSession`
- `ListServiceSessions`

现状：
- 已有 `enrichServiceSessionsWithServiceTechnicians`
- 能返回 `service_technicians`，并给每个人带 `service_start_confirmed`

建议：
- 前端展示改以 `service_technicians` 为准
- 后端此处基本不用大改
- 只需确认所有列表入口都走了 enrich

风险：**低**
- 现有接口基础较好

### 3. 自己的会话过滤 `self_only`

当前代码：`ListServiceSessions`
- `self_only` 仍是 `(technician_id = ? OR last_technician_id = ?)`

这是明显不够准确的地方。

建议：
- 改为优先按多人字段过滤
- 理想方案：
  - 命中 `service_technician_ids` 包含自己
  - 或 `start_confirmed_technician_ids` 包含自己
  - 再兼容 `technician_id / last_technician_id`

风险：**中到高**
- 涉及 JSON 字段查询
- MySQL/SQLite 测试环境写法可能不同
- 需要补测试

### 4. 超时释放逻辑

当前代码：`backend/handlers/table.go`
- 待开始超时释放仍要求 `technician_id IS NOT NULL`
- 超时后还会按旧 `technician_id` 回收考勤 busy -> idle

建议：
- 这一层短期**不要动太大**
- 因为它不是“展示问题”，而是状态机问题
- 先保留 `technician_id` 在状态机里的兼容作用

风险：**高**
- 这里若贸然改，会影响：
  - 起单超时
  - 自动回退选人
  - 技师忙闲状态回收

### 5. 在服冲突判断 / 单人忙闲判断

当前代码中不少地方仍通过 `technician_id` 判断：
- 某技师是否有进行中服务
- 某单当前归属哪个技师

建议：
- **短期不改核心状态判断**
- 保留 `technician_id` 作为状态机兼容锚点
- 只取消“展示主负责人”的语义

风险：**高**
- 涉及服务开始、结束、并发冲突、考勤占用
- 一旦误改会直接影响核心业务流

### 6. 会话创建时默认把第一个服务人员写入 `technician_id`

位置大致在：
- `service_session_factory.go`
- `ChooseServiceSessionRoom`
- `RestoreCanceledServiceSession`

建议：
- **短期可以保留**
- 因为这是兼容字段的来源之一
- 只是不再把它解释成“主负责人”

风险：**低**
- 如果仅保持兼容，不动也没问题
- 若强行改空，会连锁影响后端旧逻辑

---

## 四、推荐落地方案

### 方案 A：只取消展示语义，保留后端兼容逻辑

内容：
- 前端不再把 `session.technician` 当唯一负责人展示
- 改为展示 `service_technicians`
- “我的单 / 我的待开始 / 我的服务中”统一改为多人归属判断
- 后端字段结构不动

优点：
- 见效最快
- 风险最低
- 能立刻消除 A/B 显示错位问题

风险等级：**低到中**

**这是最推荐的方案。**

### 方案 B：展示和部分查询一起改

内容：
- 包含方案 A
- 同时改后端 `self_only` 等接口查询，逐步从 `technician_id` 转向多人字段

优点：
- 前后端语义更一致

风险等级：**中**
- 需要处理 JSON 查询兼容和测试覆盖

### 方案 C：彻底去 `technician_id` 中心化

内容：
- 业务判断、查询、统计、状态机全面改造
- `technician_id` 只剩历史迁移用途，甚至最终可废弃

风险等级：**高**
- 不适合现在直接做

---

## 五、我的最终判断

### 应不应该降级

**应该。**

### 应该怎么降级

**先降级“展示语义”，不要先删“状态机兼容作用”。**

### 风险多大

- **只改前端展示与归属判断**：低到中
- **连后端查询一起改**：中
- **动后端状态机/考勤/超时/在服判断**：高

### 最建议的落地顺序

1. 前端取消把 `technician_id` 当主负责人展示
2. 前端统一改成展示 `service_technicians` + 已扫码状态
3. 前端“我的单/我的服务中/我的待开始”改用多人归属判断
4. 后端再逐步治理 `self_only` 等查询
5. 最后才评估是否继续削弱 `technician_id` 在状态机中的作用

一句话总结：

- **应该把 `technician_id` 从“展示主负责人”降级为纯兼容字段**
- **但当前不适合直接把它从后端核心流程中拿掉**
- **最稳妥的方案是先改前端展示和查询归属口径，再逐步治理后端兼容依赖**
