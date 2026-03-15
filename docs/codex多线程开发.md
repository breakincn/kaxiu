# Codex 如何在 kabao 项目中并行开发并降低代码冲突

## 问题 1

### Codex 如何针对当前的 kabao 项目，开启多个线程任务并行开发，而又不让每个线程改动的代码产生冲突？

针对 `kabao` 这个仓库，最稳的做法不是多个线程都在同一份工作区里改，而是：

1. 每个线程一个独立 `git worktree`
2. 每个线程一个独立分支
3. 每个线程只负责一个清晰边界的子任务
4. 合并前先做串行 `rebase` 和测试

这个项目本身就适合按模块边界拆任务，因为当前结构已经分成了比较清晰的目录：

- `backend/handlers`
- `backend/models`
- `backend/routes`
- `backend/scheduler`
- `backend/queue`
- `backend/sessionflow`
- `frontend/src/api`
- `frontend/src/views`

如果你让多个线程同时动同一组文件，比如都改 `backend/routes` 或同一个 handler，冲突基本不可避免。要避免冲突，核心不是靠 Codex 自动协调，而是先把任务切成文件集几乎不重叠的几块。

## 推荐工作流

先从主分支拉出多个 `worktree`，不要让多个 Codex 都指向同一个目录：

```bash
git worktree add ../kabao-task-api -b codex/api-cleanup
git worktree add ../kabao-task-scheduler -b codex/scheduler-retry
git worktree add ../kabao-task-frontend -b codex/frontend-order-page
```

然后分别在不同目录启动不同 Codex 线程：

- `/Users/will/Projects/Go/kabao-task-api`
- `/Users/will/Projects/Go/kabao-task-scheduler`
- `/Users/will/Projects/Go/kabao-task-frontend`

这样做的好处：

- 每个线程有独立文件系统视图，不会互相覆盖本地改动
- 每个线程在独立分支上提交，便于 review 和回滚
- 就算都在改同一个仓库，也不会因为一个线程未提交的改动污染另一个线程

## 在 kabao 里怎么拆任务最不容易冲突

优先按“功能切片”拆，不要按“分层”拆。

适合的拆法：

- 线程 A：只做某个订单或核销相关接口，改它自己的 `models + handlers + routes + tests`
- 线程 B：只做某个调度器逻辑，改 `backend/scheduler` 及其直接依赖
- 线程 C：只做某个商户页面，改 `frontend/src/views/merchant` 和对应 `frontend/src/api`
- 线程 D：只做文档、脚本或数据修复工具

不适合的拆法：

- A 改 handler
- B 改 route
- C 改 model
- 但三者服务于同一个需求

这种拆法最后通常会汇聚到同一批接口定义、结构体字段和返回值格式，冲突概率很高。

更好的原则是：

- 一个需求，一个线程
- 不要一个层，一个线程

例如：

- “新增 merchant 结算报表接口”
  - 同一线程改 `models + handlers + routes + tests`
- “重构 scheduler 重试逻辑”
  - 同一线程改 `scheduler + queue + config`

## 给 Codex 的任务描述要带边界

每个线程开工时，提示词里直接写清楚约束，例如：

```text
只修改 backend/scheduler 及其直接测试，不要改 handlers/routes/models。
如果发现必须跨目录改动，先停下并列出需要变更的文件。
```

或者：

```text
仅在 frontend/src/views/merchant 和 frontend/src/api/merchant.ts 内工作。
不要修改共享组件、全局路由或构建配置。
```

这样可以显著降低线程顺手去改公共文件的概率。

## 合并顺序

不要让所有线程最后一起硬合。建议顺序：

1. 先合底层公共改动
2. 再让后续线程 `rebase`
3. 最后合依赖上游接口的前端或页面改动

例如，如果某个线程改了 `backend/models` 的公共结构体，它就应该先合，其他依赖这个结构体的线程再同步。

## 一套可直接执行的规范

你可以在 `kabao` 项目里固定用这套规则：

- 分支名统一 `codex/<task>`
- 每个线程一个 `git worktree`
- 每个线程只允许改一组预先声明的目录或文件
- 涉及共享文件时，指定一个主线程负责
- 合并前只跑对应范围测试
- 每个线程先提交，再 `rebase`，最后再合并

示例测试范围：

```bash
cd backend && go test ./handlers
cd backend && go test ./scheduler
cd frontend && npm run build
```

## 问题 2

### 继续

针对当前 `kabao` 仓库，可以直接落地成下面这套 5 线程并行模板。这个模板的目标不是把所有目录平均分，而是尽量避免共享文件碰撞。

## kabao 并行线程模板

### 线程 1：调度器链路

负责范围：

- `backend/scheduler`
- `backend/queue`
- `backend/sessionflow`

适合任务：

- 超时推进
- 自动叫号
- 会话状态流转
- 调度补偿和重试

禁止改动：

- `backend/routes`
- 大范围 `backend/handlers`
- 前端文件

### 线程 2：商户后台接口

负责范围：

- `backend/handlers` 中 merchant 相关 handler
- `backend/models` 中 merchant 相关模型
- 必要时改对应 `backend/routes`

适合任务：

- 商户配置
- 排队管理接口
- 核销后台接口

限制：

- 如果要改公共模型，先确认是否会影响线程 1 或线程 3

### 线程 3：用户端或商户端页面

负责范围：

- `frontend/src/views/user`
- `frontend/src/views/merchant`
- `frontend/src/api`

适合任务：

- 页面交互
- 列表与详情页
- 页面内 API 对接

禁止改动：

- `frontend/src/components` 公共组件，除非它被明确指定归这个线程负责
- `frontend/src/router`，除非本线程就是“路由和入口调整”

### 线程 4：平台管理或公共配置

负责范围：

- `frontend/src/views/platformAdmin`
- `backend/config`
- `backend/middleware`
- 少量平台管理接口

适合任务：

- 平台管理配置
- 权限和中间件
- 平台后台页面

注意：

- 这类改动经常带全局副作用，应该尽量单独成线程，避免与业务线程混改

### 线程 5：文档、脚本、迁移

负责范围：

- `docs`
- `backend/scripts`
- `backend/migrations`
- `backend/cmd` 下独立工具

适合任务：

- 数据修复脚本
- 运维脚本
- 文档和排查报告
- 数据库迁移

## 线程分配原则

在 `kabao` 里，最容易冲突的文件通常是：

- `backend/routes/*`
- `backend/models/*` 里的公共结构体
- `frontend/src/router/*`
- `frontend/src/components/*`
- 全局配置类文件

因此建议：

1. 公共入口文件只指定一个线程负责
2. 多个线程如果都依赖公共结构体，先由一个基础线程完成公共改动
3. 后续线程只在它的 `worktree` 中 `rebase` 最新主线，不要各自独立发散太久

## 建议的实际执行步骤

### 1. 先为任务建独立 worktree

```bash
git worktree add ../kabao-scheduler -b codex/scheduler-task
git worktree add ../kabao-merchant-api -b codex/merchant-api-task
git worktree add ../kabao-merchant-web -b codex/merchant-web-task
git worktree add ../kabao-platform -b codex/platform-task
```

### 2. 每个线程启动前写清楚边界

示例：

```text
你只允许修改 backend/scheduler、backend/queue、backend/sessionflow。
如果必须修改 models 或 routes，先停止并列出原因，不要直接改。
```

### 3. 每个线程完成后先做本地验证

示例：

```bash
cd backend && go test ./scheduler
cd backend && go test ./handlers
cd frontend && npm run build
```

### 4. 按依赖顺序合并

推荐顺序：

1. 公共模型或配置
2. 后端业务接口
3. 调度或异步流程
4. 前端页面
5. 文档和脚本

## 一句话结论

Codex 在 `kabao` 里要并行开发且尽量不冲突，关键不是简单多开几个线程，而是：

- 多开 `git worktree`
- 每个线程绑定独立分支
- 每个线程绑定明确文件边界
- 公共文件指定唯一负责人
- 按依赖顺序合并

只要你按这个规则执行，Codex 多线程开发会比在同一工作区里并行修改稳定很多。
