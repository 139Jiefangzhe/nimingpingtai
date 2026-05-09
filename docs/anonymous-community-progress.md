# 企业微信匿名问答 + 帖子交流社区 二开进度

更新时间：2026-05-09

## 当前状态

- Apache Answer 匿名社区二开已进入“本地可预览”阶段，基线提交仍为 `fca80abb`。
- 预发环境已完成远端部署与在线迁移，当前可通过正式预发域名访问：
  - 地址：`https://forum.xingyuanjituan.cn/community`
  - 远端主机：`47.94.135.253`
  - 预发目录：`/opt/niming-community-predeploy`
  - 入口链路：`cloudflared -> caddy -> 127.0.0.1:9080 -> answer-app`
- 当前运行中的预发镜像标签：
  - `niming-answer-app:hotfix-20260509-193502-c8d7a536-voteid`
  - `niming-vault-service:predeploy-20260507-00560492-communitytags`
- 本轮已完成“普通登录用户去除声望门槛”的后端、前端、默认配置和存量数据迁移：
  - 数据库版本已从 `34` 升级到 `35`
  - 新增迁移版本：`v1.8.4`
  - 普通用户操作相关 `rank.*` 配置已被迁移为 `0`
- 当前已经可以在本机 Docker 环境直接打开匿名社区预览：
  - 地址：`http://127.0.0.1:9080/community`
  - 预览模式：`COMMUNITY_PREVIEW_MODE=local`
  - 工作方式：不依赖企业微信，使用本地预置匿名身份与种子数据
  - 当前状态：镜像已按最新源码重建，容器不再依赖手工替换二进制
- 本轮已完成：
  - 匿名社区后端主干
  - 匿名社区前端 H5 页面
  - 本地预览登录/种子数据/路由接线
  - Docker 本地预览编排
  - 本地 Docker 构建链固化
- 已完成的验证：
  - Docker Go `1.24` 环境下 `go test ./internal/service/community ./internal/controller ./internal/router ./cmd` 通过
  - Node `20` 容器内 `pnpm build` 通过
  - 本地容器已启动成功，`/community`、`/answer/api/v1/community/preview/bootstrap`、`/answer/api/v1/community/preview/login` 可正常访问
  - 使用最新源码重建 `deploy-answer-app:latest` 成功并完成容器重建
  - `discussion` / `qa` 列表接口返回匿名种子数据正常
  - 匿名预览账号发帖接口验证通过
- 当前访问说明已确认：
  - 在当前 `WSL2` 环境中，容器内可直接访问地址为 `http://172.27.213.19:9080/community`
  - 宿主机局域网 IP 为 `10.7.1.161`
  - 若要通过 `http://10.7.1.161:9080/community` 访问，仍需在 Windows 管理员权限下额外配置 `portproxy` 与防火墙放行

## 2026-05-09 进展补充

### 1. 评论“未找到”问题已修复并发布预发

- 根因定位：
  - 社区详情接口返回的回复 ID 是原始 17 位 ID，例如 `10020000000000144`
  - 原始回复 ID 传给 `uid.DeShortID` 不会出错
  - 真正触发“评论未找到”的边界是空的可选 `reply_comment_id` 被解码成 `"0"`，导致后端按评论 ID `0` 查询并返回 not found
- 后端修复：
  - `CreateComment` 中回复 ID 改为直接使用 `ctx.Param("answerId")`
  - `reply_comment_id` 和分页查询用的 `comment_id` 仅在非空时才执行短 ID 解码
  - 新增 `decodeOptionalCommunityID`，保留空值为空字符串
- 关键文件：
  - `internal/controller/community_controller.go`

### 2. 社区详情页已增加投票入口

- 前端已在社区详情页增加轻量投票按钮：
  - 问题标题右侧支持 ▲ / ▼
  - 每条回复下方支持 ▲ / ▼
  - 成功后优先使用接口返回的 `votes`，否则按方向做本地增减
- 投票仍复用 Apache Answer 原有接口：
  - `/answer/api/v1/vote/up`
  - `/answer/api/v1/vote/down`
- 关键文件：
  - `ui/src/pages/Community/Detail/index.tsx`
- 注意：
  - 投票接口仍会走现有登录、权限和验证码频率策略
  - 本次只新增社区版 UI 入口，没有绕过后端投票风控

### 3. 预发热发布已完成

- 新镜像：
  - `niming-answer-app:hotfix-20260509-174601-8a243edb-commentvote`
- 上一版本：
  - `niming-answer-app:hotfix-20260508-165453-e4908275-navtags`
- 远端主机：
  - `47.94.135.253`
- 远端目录：
  - `/opt/niming-community-predeploy`
- 本次只重建：
  - `answer-app`
- 未重建：
  - `vault-service`
  - 数据库
- 远端发布记录：
  - `/opt/niming-community-predeploy/release-hotfix-20260509-174601-8a243edb-commentvote.txt`
- 远端 `.env` 备份：
  - `/opt/niming-community-predeploy/.env.bak.20260509-183058.hotfix-20260509-174601-8a243edb-commentvote`
- 回滚脚本：
  - `/opt/niming-community-predeploy/rollback-hotfix-20260509-174601-8a243edb-commentvote.sh`

### 4. 本次验证结果

- 本地验证：
  - `git diff --check` 通过
  - `go build ./internal/...` 通过
  - `cd ui && npx tsc --noEmit` 通过
  - `cd ui && pnpm build` 通过
- 镜像验证：
  - Docker 镜像构建成功，大小约 `201MB`
  - 镜像二进制包含 `已点赞`
  - 镜像二进制包含 `decodeOptionalCommunityID`
- 预发验证：
  - `https://forum.xingyuanjituan.cn/community/questions/10010000000000143` 返回 `200`
  - `https://forum.xingyuanjituan.cn/answer/api/v1/questions/10010000000000143` 返回 `200`
  - API 返回回复原始 ID：`10020000000000144`
  - 未登录评论接口返回 `401 Unauthorized`，不再表现为“未找到”
  - 最近 5 分钟 `answer-app` 日志未出现 `comment.not_found`、`CommentNotFound`、`object.not_found` 或新错误

### 5. 回滚命令

```bash
ssh root@47.94.135.253 'cd /opt/niming-community-predeploy && ./rollback-hotfix-20260509-174601-8a243edb-commentvote.sh'
```

### 6. 投票 ObjectID 原始 ID 兼容修复已发布

- 根因定位：
  - 社区详情页投票按钮传给 `/answer/api/v1/vote/up` 和 `/answer/api/v1/vote/down` 的是原始 17 位对象 ID
  - 原有 `VoteController` 无条件执行 `uid.DeShortID`，会把社区原始 ID 误解码，导致投票落到错误对象或返回失败
- 后端修复：
  - `VoteUp` 和 `VoteDown` 统一改为 `decodeVoteObjectID`
  - 短 ID 仍正常解码为原始 ID
  - 原始 ID、空 ID、解码结果为空或 `"0"` 时保留原值
- 关键文件：
  - `internal/controller/vote_controller.go`
  - `internal/controller/vote_controller_test.go`
- 新镜像：
  - `niming-answer-app:hotfix-20260509-193502-c8d7a536-voteid`
- 上一版本：
  - `niming-answer-app:hotfix-20260509-174601-8a243edb-commentvote`
- 本次只重建：
  - `answer-app`
- 未重建：
  - `vault-service`
  - 数据库
- 远端发布记录：
  - `/opt/niming-community-predeploy/release-hotfix-20260509-193502-c8d7a536-voteid.txt`
- 远端 `.env` 备份：
  - `/opt/niming-community-predeploy/.env.bak.20260509-200726.hotfix-20260509-193502-c8d7a536-voteid`
- 回滚脚本：
  - `/opt/niming-community-predeploy/rollback-hotfix-20260509-193502-c8d7a536-voteid.sh`
- 本地验证：
  - `go test ./internal/controller -run TestDecodeVoteObjectID -v` 通过
  - `go build ./internal/...` 通过
  - `go build ./...` 通过
  - `git diff --check` 通过
- 预发验证：
  - 当前远端 `DEPLOY_TAG=hotfix-20260509-193502-c8d7a536-voteid`
  - `answer version 2.0.0`，`revision: c8d7a536`
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `https://forum.xingyuanjituan.cn/answer/api/v1/questions/10010000000000143` 返回 `200`
  - 未登录用原始 ID 调用投票接口返回 `401 Unauthorized`，未触发对象不存在或 ID 解码错误
  - 最近 5 分钟 `answer-app` 日志未出现 `vote`、`object.not_found`、`not_found`、`panic`、`fatal`、`error`
- 回滚命令：

```bash
ssh root@47.94.135.253 'cd /opt/niming-community-predeploy && ./rollback-hotfix-20260509-193502-c8d7a536-voteid.sh'
```

## 2026-05-07 进展补充

### 1. `/community` 已切换为正式企微授权入口

- 前端已移除本地 preview bootstrap/login 依赖：
  - `ui/src/pages/Community/Layout/index.tsx`
- 未登录用户点击“进入匿名社区”后，会直接跳转：
  - `/answer/api/v1/wecom/auth/start?return_to=<当前 community 路径>`
- 页面标题副文案已从“本地预览版”改为“企业微信登录版”。
- 本地 preview 服务代码仍保留在仓库中，但预发入口链路已不再使用 `COMMUNITY_PREVIEW_MODE=local`。

### 2. 企微授权回跳已改为社区优先

- 企微 `AuthCallback` 默认回跳已从 `/home?tab=discussion` 改为 `/community`：
  - `internal/service/wecom/service.go`
- `/users/auth-landing` 已补上 `return_to` 处理：
  - `ui/src/pages/Users/AuthCallback/index.tsx`
  - `ui/src/utils/guard.ts`
- 已在预发环境验证：
  - 不传 `return_to` 时，企微授权 `state` 默认解码为 `/community`
  - 显式传 `return_to=/community/qa?tab=new` 时，`state` 可正确透传

### 3. 企微成员事件与 Vault 状态同步已接线

- 企微回调已支持 `change_contact` 事件：
  - `create_user`
  - `update_user`
  - `delete_user`
- Vault 已新增：
  - `/internal/identity/update-status`
- 相关代码位置：
  - `internal/service/wecom/callback_service.go`
  - `internal/schema/wecom_schema.go`
  - `internal/vaultapp/server.go`
- 当前实现会：
  - 创建/更新成员时刷新 Vault 身份映射
  - 删除成员时把匿名身份状态切到 `disabled`
  - 同步写入 `audit_reveal_log`，来源标记为 `wecom_event`

### 4. 2026-05-07 预发发布已完成

- 远端主机：`47.94.135.253`
- 预发目录：`/opt/niming-community-predeploy`
- 当前线上镜像：
  - `niming-answer-app:predeploy-20260507-b47d1802-wecomjson`
  - `niming-vault-service:predeploy-20260507-b47d1802-wecomjson`
- 当前远端 `.env` 已生效：
  - `WECOM_DEFAULT_RETURN_TO=/community`
  - `VAULT_BASE_URL=http://vault-service:8091`
  - `COMMUNITY_PREVIEW_MODE` 已移除
- 当前容器状态：
  - `answer-app` healthy
  - `vault-service` healthy
- 本轮没有新增数据库 schema 迁移版本，属于应用逻辑与预发配置切换。

### 5. 本轮回滚资源

- 环境备份：
  - `/opt/niming-community-predeploy/.env.bak.20260507-152236.predeploy-20260507-b47d1802-wecomjson`
- 回滚脚本：
  - `/opt/niming-community-predeploy/rollback-predeploy-20260507-b47d1802-wecomjson.sh`
- 发布记录：
  - `/opt/niming-community-predeploy/release-predeploy-20260507-b47d1802-wecomjson.txt`

### 6. 本次 `b47d1802` 发布结果

- 本次发布对应 GitHub 提交：
  - `b47d1802 fix: support json clients for wecom auth start`
- 实际部署切换结果：
  - 旧 `DEPLOY_TAG`：`predeploy-20260507-b5a943aa-wecom302`
  - 新 `DEPLOY_TAG`：`predeploy-20260507-b47d1802-wecomjson`
  - 仅重建服务：`answer-app`、`vault-service`
- 已验证：
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `GET /answer/api/v1/wecom/auth/start?return_to=/community` 返回 `302`
  - `Location` 头直出正式企微 `https://open.weixin.qq.com/connect/oauth2/authorize?...`
  - `Accept: application/json` 请求返回 JSON：`{\"data\":{\"authorization_url\":...}}`
  - 远端 `answer-app` 与 `vault-service` 均为 `healthy`

### 7. 本次 `00560492` 发布结果

- 本次发布对应 GitHub 提交：
  - `8a777738 feat: notify community authors via wecom`
  - `00560492 fix: drop unused ask page card import`
- 已实现帖子被回复时自动通知发帖人：
  - 社区回复创建后异步触发企微应用消息推送
  - `answer-app` 通过 Vault lookup 获取发帖人的企微 `user_id`
  - Vault 新增 `/internal/identity/lookup` 内部接口
  - 企微发送 Markdown 应用消息，内容包含帖子标题、回复人、摘要和回看链接
- 本次同时保留 `/questions/add` 页面右侧“如何排版”帮助面板删除，并补掉未使用 `Card` import，保证生产构建通过。
- 实际部署切换结果：
  - 新 `DEPLOY_TAG`：`predeploy-20260507-00560492-wecompush`
  - 仅重建服务：`answer-app`、`vault-service`
  - 未执行新增 SQL 或数据库版本迁移
- 本轮回滚资源：
  - 环境备份：`/opt/niming-community-predeploy/.env.bak.20260507-165121.predeploy-20260507-00560492-wecompush.from-current`
  - 回滚脚本：`/opt/niming-community-predeploy/rollback-predeploy-20260507-00560492-wecompush.sh`
  - 发布记录：`/opt/niming-community-predeploy/release-predeploy-20260507-00560492-wecompush.txt`
- 已验证：
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `GET /answer/api/v1/wecom/auth/start?return_to=/community` 返回 `302`
  - `Accept: application/json` 请求返回 JSON `authorization_url`
  - Vault `/internal/identity/lookup` 已生效，空参数请求按预期返回 `400`
  - 远端 `answer-app` 与 `vault-service` 均为 `healthy`

### 8. 固定问题标签白名单预发发布结果

- 本次发布对应 GitHub 提交：
  - `6a020c18 feat: enforce fixed question tag whitelist`
  - `4a524a55 feat: unify fixed tags on question edit page`
  - `00560492 fix: drop unused ask page card import`
- 已实现：
  - `/questions/add` 创建页标签输入改为固定四标签按钮：`交流`、`问答`、`投诉建议`、`经验分享`
  - 编辑页同样收敛到固定四标签，并过滤历史非白名单标签
  - 提交前前端再次过滤非白名单标签
  - 后端 `QuestionParams.ValidateTags()` 校验空标签和非法标签
  - `AddQuestion` / `UpdateQuestion` 均接入后端白名单校验
- 实际部署切换结果：
  - 当前 `DEPLOY_TAG`：`predeploy-20260507-00560492-communitytags`
  - 上一运行 tag：`predeploy-20260507-00560492-wecompush`
  - 仅重建服务：`answer-app`、`vault-service`
  - 未执行新增 SQL 或数据库版本迁移
- 发布过程备注：
  - 曾尝试发布 `predeploy-20260507-8a777738-communitytags`，但该镜像缺少完整 UI embed，远端日志报 `open build/index.html: file does not exist`
  - 已立即使用回滚脚本恢复旧版本，确认 `/community` 恢复 `200`
  - 最终改用 GitHub 最新提交 `00560492` 对应的完整镜像重新标记并发布
- 本轮回滚资源：
  - 环境备份：`/opt/niming-community-predeploy/.env.bak.20260507-165840.predeploy-20260507-00560492-communitytags`
  - 回滚脚本：`/opt/niming-community-predeploy/rollback-predeploy-20260507-00560492-communitytags.sh`
  - 发布记录：`/opt/niming-community-predeploy/release-predeploy-20260507-00560492-communitytags.txt`
- 已验证：
  - 远端 `answer-app` 与 `vault-service` 均为 `healthy`
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `GET /answer/api/v1/wecom/auth/start?return_to=/community` 返回 `302`
  - `Location` 头直出正式企微 `https://open.weixin.qq.com/connect/oauth2/authorize?...`
  - 固定标签前端 chunk `/static/js/5088.314fe22d.chunk.js` 已包含四个白名单标签
  - 固定标签前端 chunk 中未再出现 `TagSelector` / `maxTagLength` 自由输入标识
- 未完成验证：
  - API 非法标签绕过测试需要有效登录态或 Bearer Token，本轮未伪造线上用户凭据，因此未直接调用创建问题 API

## 2026-05-08 进展补充

### 1. 匿名社区发布页标签示例文案热修复

- 本次发布对应 GitHub 提交：
  - `6c23b02d fix: remove community compose tag examples`
- 已修改：
  - `ui/src/pages/Community/Compose/index.tsx`
  - 删除匿名问答/匿名讨论发布页的英文示例标签文案 `示例：Culture, Workflow, Product`
  - 发布页标签输入默认值从 `Culture, Workflow` 改为空字符串
  - 标签区域提示改为 `请选择一个或多个标签`
- 实际部署切换结果：
  - 新 `DEPLOY_TAG`：`hotfix-20260508-150342-6c23b02d-composehint`
  - 上一运行 tag：`predeploy-20260507-00560492-communitytags`
  - 仅热替换服务：`answer-app`
  - 未重建 `vault-service`，未停止 PostgreSQL/Redis/Vault
  - 未执行新增 SQL 或数据库版本迁移
- 本轮回滚资源：
  - 环境备份：`/opt/niming-community-predeploy/.env.bak.20260508-151814.hotfix-20260508-150342-6c23b02d-composehint`
  - 回滚脚本：`/opt/niming-community-predeploy/rollback-hotfix-20260508-150342-6c23b02d-composehint.sh`
  - 发布记录：`/opt/niming-community-predeploy/release-hotfix-20260508-150342-6c23b02d-composehint.txt`
- 已验证：
  - 本地 Docker 镜像构建成功：`niming-answer-app:hotfix-20260508-150342-6c23b02d-composehint`
  - 远端 `answer-app` 切换后为 `healthy`
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `https://forum.xingyuanjituan.cn/community/questions/new` 返回 `200`
  - `https://forum.xingyuanjituan.cn/community/discussions/new` 返回 `200`
  - 远端容器二进制中已包含 `请选择一个或多个标签`
  - 远端容器二进制中未再出现 `示例：Culture, Workflow, Product` 或默认 `Culture, Workflow`

### 2. 匿名社区导航合并与社区标签校验热修复

- 本次发布对应 GitHub 提交：
  - `e4908275 feat: merge community nav and fix compose tags`
- 已修改：
  - `ui/src/pages/Community/Layout/index.tsx`
  - `ui/src/pages/Community/Feed/index.tsx`
  - `ui/src/pages/Community/Compose/index.tsx`
  - `internal/schema/question_schema.go`
- 已实现：
  - 社区顶部导航只保留 `交流`、`问答`、管理员可见的 `管理`
  - `+ 发讨论` / `+ 发问答` 移到对应列表页排序按钮前
  - 社区发布页标签输入改为固定四标签按钮：`交流`、`问答`、`投诉建议`、`经验分享`
  - 社区问答频道标签必填，未选标签时前端禁用提交
  - 社区讨论频道标签可选，允许空标签
  - 后端 `QuestionAdd.ValidateTags()` 已按 `ChannelType` 区分：`discussion` 允许空标签，`qa` 必填；非空标签均必须在白名单内
  - 常规问答编辑与自问自答仍按 `qa` 规则要求至少一个白名单标签
- 实际部署切换结果：
  - 新 `DEPLOY_TAG`：`hotfix-20260508-165453-e4908275-navtags`
  - 上一运行 tag：`hotfix-20260508-150342-6c23b02d-composehint`
  - 仅热替换服务：`answer-app`
  - 未重建 `vault-service`，未停止 PostgreSQL/Redis/Vault
  - 未执行新增 SQL 或数据库版本迁移
- 本轮回滚资源：
  - 环境备份：`/opt/niming-community-predeploy/.env.bak.20260508-170810.hotfix-20260508-165453-e4908275-navtags`
  - 回滚脚本：`/opt/niming-community-predeploy/rollback-hotfix-20260508-165453-e4908275-navtags.sh`
  - 发布记录：`/opt/niming-community-predeploy/release-hotfix-20260508-165453-e4908275-navtags.txt`
- 已验证：
  - `npx tsc --noEmit` 通过
  - `pnpm build` 通过
  - `go build ./internal/...` 通过
  - 本地 Docker 镜像构建成功：`niming-answer-app:hotfix-20260508-165453-e4908275-navtags`
  - 远端 `answer-app` 切换后为 `healthy`
  - `https://forum.xingyuanjituan.cn/community` 返回 `200`
  - `https://forum.xingyuanjituan.cn/community/qa` 返回 `200`
  - `https://forum.xingyuanjituan.cn/community/questions/new` 返回 `200`
  - `https://forum.xingyuanjituan.cn/community/discussions/new` 返回 `200`
  - 远端容器二进制中已包含 `+ 发问答`、`+ 发讨论`、四个固定标签和 `请选择一个或多个标签`
  - 远端容器二进制中未再出现 `Culture, Workflow`

## 2026-04-30 进展补充

### 1. 普通用户声望限制已按白名单方式移除

- 已完成普通登录用户常规动作的声望门槛放开，但没有放开管理员/版主类动作。
- 当前已放开的普通动作包括：
  - 提问
  - 回答
  - 评论
  - 举报
  - 新增标签
  - 问题/回答/评论投票
  - 邀请回答
  - `link.url_limit`
- 当前仍保留限制的高风险动作包括：
  - 审核
  - 关闭 / 重开
  - 保留标签
  - 标签同义词
  - 跨用户编辑类动作
- 实现策略不是简单把 `checkUserRank()` 改成永远返回 true，而是只对白名单普通动作替换最后一步的 `rank` 门槛判断，保留：
  - 登录要求
  - 角色 power
  - 对象所有权校验
  - 验证码和风控
  - 声望记录本身

### 2. 声望改造已覆盖默认配置、后台配置和已有数据库

- 已修改后端权限核心：
  - `internal/service/rank/rank_service.go`
  - `internal/controller/permission_controller.go`
- 已修改默认权限阈值与后台最小值约束：
  - `internal/base/constant/privilege.go`
  - `internal/migrations/init_data.go`
  - `internal/schema/siteinfo_schema.go`
  - `ui/src/pages/Admin/Privileges/index.tsx`
  - `i18n/en_US.yaml`
  - `i18n/zh_CN.yaml`
- 已新增已有数据库迁移：
  - `internal/migrations/v34.go`
  - `internal/migrations/v34_test.go`
  - `internal/migrations/migrations.go`
- 已补充最小测试覆盖：
  - `internal/service/rank/rank_service_test.go`
- 已验证：
  - `go build ./internal/...`
  - `cd ui && npx tsc --noEmit`
  - `go test ./internal/service/siteinfo_common -v`
  - `go test ./internal/repo/repo_test -run Test_siteInfoRepo_SaveByType -v`
  - `go test ./internal/migrations -v`
  - `go test ./internal/service/rank -v`

### 3. 预发部署、迁移和回滚机制已落地

- 已在远端预发环境完成镜像构建、镜像传输、单次迁移执行和正式容器切换。
- 上线顺序为：
  1. 远端备份当前 `.env`
  2. 生成回滚脚本
  3. 传输新 `answer-app` 镜像到远端
  4. 用临时 `.env` 单独执行 `answer upgrade`
  5. 切换正式 `DEPLOY_TAG`
  6. 重建预发容器并做健康检查
- 已验证的运行状态：
  - `/healthz` 正常
  - `/community` 正常
  - `/answer/api/v1/wecom/auth/start` 返回 `200`
  - `answer-app` 和 `vault-service` 均为 `healthy`
- 当前线上迁移结果：
  - `version.version_number = 35`
  - 普通动作相关 `rank.*` 配置均为 `0`
- 已生成的回滚资源：
  - 环境备份：`/opt/niming-community-predeploy/.env.bak.20260430-105021.predeploy-20260430-fca80abb-rankperm`
  - 回滚脚本：`/opt/niming-community-predeploy/rollback-predeploy-20260430-fca80abb-rankperm.sh`
  - 发布记录：`/opt/niming-community-predeploy/release-predeploy-20260430-fca80abb-rankperm.txt`

### 4. 非结构化文件存储现状已梳理

- 当前项目的图片/附件不存入 PostgreSQL 二进制字段，而是落到应用数据卷：
  - 容器内路径：`/data/uploads`
  - 宿主机路径：`/var/lib/docker/volumes/niming-community-predeploy_answer-data/_data/uploads`
- 当前目录结构包括：
  - `avatar`
  - `avatar_thumb`
  - `post`
  - `files/post`
  - `branding`
  - `deleted`
- 数据库仅保存元信息，表为 `file_record`，核心字段为：
  - `file_path`
  - `file_url`
  - `object_id`
  - `source`
- 当前预发环境的真实状态：
  - `uploads` 目录结构已存在
  - 当前目录下实际文件数为 `0`
  - 当前 `file_record` 表暂无上传记录
- 当前高级配置下：
  - 图片扩展名允许：`jpg/jpeg/png/gif/webp`
  - 附件扩展名白名单为空数组
- 结合代码逻辑，当前环境结论是：
  - 图片上传可用
  - 通用附件默认不可上传
  - 视频也不会被直接作为附件上传

### 5. 现阶段不建议引入 SeaweedFS

- 目前仍是单机预发部署，`answer-app` 只有一台，上传目录走单个持久卷，且当前文件量几乎为零。
- 在这个阶段直接引入 SeaweedFS 的收益很低，运维复杂度会明显增加。
- 当前更合理的顺序应为：
  1. 先补宿主机级备份
  2. 明确是否开放附件/视频上传
  3. 等需要多实例共享文件或开始出现较大附件规模时，再评估对象存储或分布式存储
- 代码层面已经保留了后续切外部存储的扩展点：
  - `plugin/storage.go`
  - `internal/service/uploader/upload.go`

## 已完成实现

### 1. 匿名社区数据模型

- 已扩展 `question` 表字段：
  - `channel_type`
  - `visibility_mode`
  - `moderation_state`
- 已新增实体：
  - `internal/entity/anonymous_profile_entity.go`
  - `internal/entity/moderation_action_entity.go`
- 已补充迁移与初始化表注册：
  - `internal/migrations/v32.go`
  - `internal/migrations/migrations.go`
  - `internal/migrations/init_data.go`

### 2. 问题/帖子双频道模型

- 已在问题实体与 schema 中加入频道和匿名可见性字段：
  - `qa`
  - `discussion`
- 已修改问题展示格式化链路：
  - `internal/schema/question_schema.go`
  - `internal/service/question_common/question.go`
- 已修改提问创建逻辑：
  - 讨论频道自动放宽最少标签和推荐标签要求
  - 新建内容默认写入匿名可见性与正常审核状态
  - 讨论贴允许标题缺省，由社区层生成持久化标题
- 已修复 `UserCenterLoginService.ExternalLogin(...)` 中新外部登录路径的空指针风险：
  - `internal/service/user_external_login/user_center_login_service.go`

### 3. 社区 API 主干

- 已新增社区 schema：
  - `internal/schema/community_schema.go`
- 已新增社区 service：
  - `internal/service/community/service.go`
- 已新增社区 controller：
  - `internal/controller/community_controller.go`
- 已注册社区路由：
  - `internal/router/answer_api_router.go`
- 已接入依赖提供者：
  - `internal/service/provider.go`
  - `internal/controller/controller.go`
  - `cmd/wire_gen.go`

当前已落地的社区接口包括：

- `GET /answer/api/v1/home`
- `GET /answer/api/v1/questions/:id`
- `GET /answer/api/v1/discussions/:id`
- `POST /answer/api/v1/questions`
- `POST /answer/api/v1/discussions`
- `POST /answer/api/v1/content/:questionId/replies`
- `POST /answer/api/v1/replies/:answerId/comments`
- `POST /answer/api/v1/reports`
- `POST /answer/api/v1/admin/moderation/actions`
- `POST /answer/api/v1/admin/audit/reveal`
- `GET /answer/api/v1/community/preview/bootstrap`
- `POST /answer/api/v1/community/preview/login`
- `GET /answer/api/v1/replies/:answerId/comments`

### 4. 社区入口的权限与风控补齐

- 已为匿名社区新增接口补上与原生问答接口一致的基础门禁：
  - 提问/发帖权限校验
  - 回帖权限校验
  - 评论权限校验
  - 举报权限校验
  - 重复提交拦截
  - 验证码接入
- 已在社区请求 schema 中补充：
  - `captcha_id`
  - `captcha_code`
- 已修复社区 service 里“只转请求不执行 `Check()`”的问题：
  - 现在会为 `QuestionAdd / AnswerAddReq / AddCommentReq` 生成 HTML / ParsedText

### 5. 本地预览前端已接线

- 已新增匿名社区前端页面：
  - `ui/src/pages/Community/Layout`
  - `ui/src/pages/Community/Feed`
  - `ui/src/pages/Community/Detail`
  - `ui/src/pages/Community/Compose`
  - `ui/src/pages/Community/Moderation`
- 已新增社区前端请求封装：
  - `ui/src/services/client/community.ts`
- 已注册前端路由：
  - `/community`
  - `/community/qa`
  - `/community/discussions/new`
  - `/community/questions/new`
  - `/community/discussions/:id`
  - `/community/questions/:id`
  - `/community/moderation`
- 已实现的前端能力：
  - 本地预览启动检测
  - 一键进入匿名身份
  - 匿名头像 / 匿名昵称渲染
  - 问答 / 讨论双频道浏览
  - 帖子详情、回复、楼中评论展示
  - 发帖 / 回帖基本交互
  - 管理台基础隐藏/恢复/删除操作界面

### 6. 企业微信接入骨架

- 已新增企业微信 schema：
  - `internal/schema/wecom_schema.go`
- 已新增企业微信 service：
  - `internal/service/wecom/service.go`
- 已新增企业微信 controller：
  - `internal/controller/wecom_controller.go`
- 已注册企业微信路由：
  - `GET /answer/api/v1/wecom/auth/start`
  - `GET /answer/api/v1/wecom/auth/callback`
  - `ANY /answer/api/v1/wecom/callback`

当前企业微信侧已实现：

- OAuth 跳转地址生成
- 通过企微接口换取应用访问 token
- 通过授权 `code` 获取企业内 `userid`
- 读取员工基础资料
- 调用 Vault 解析匿名身份
- 调用现有用户中心登录服务，创建或登录匿名站内账号
- 同步匿名档案到主业务库的 `anonymous_profile`
- URL 校验签名验证
- 回调 `echostr` AES 解密
- 回调消息签名校验与基础 XML 解密解析

### 7. Vault 独立服务骨架

- 已新增 Vault HTTP 服务：
  - `internal/vaultapp/server.go`
- 已新增独立入口：
  - `cmd/vault/main.go`
- 已提供独立容器与示例编排：
  - `Dockerfile.vault`
  - `deploy/docker-compose.community.yml`

当前 Vault 服务已实现的接口：

- `POST /internal/identity/resolve`
- `POST /internal/identity/status`
- `POST /internal/identity/reveal`
- `POST /internal/audit/log`
- `GET /healthz`

当前 Vault 已具备的能力：

- `corp_id + user_id -> anon_subject_id` 映射生成
- 映射密文存储
- reveal 审计日志落库
- 共享令牌校验

### 8. 本地 Docker 预览链已固化

- 已新增 `.dockerignore`，排除以下本地构建噪音目录：
  - `.git`
  - `.github`
  - `.vscode`
  - `ui/node_modules`
  - `ui/build`
  - `build`
  - `coverage`
  - `dist`
  - `tmp`
- 已修正 `Dockerfile`：
  - 镜像构建阶段先执行 `make ui`
  - 再执行 `make clean build`
  - 避免依赖宿主机现成的 `ui/build`
- 已验证新镜像可直接重新创建预览容器，并保留匿名社区预览能力

### 9. 企业微信接入所需配置已梳理

- 当前用户手头已有：
  - `WECOM_CORP_ID`
  - `WECOM_AGENT_ID`
  - `WECOM_APP_SECRET`
- 按当前代码，企业微信正式接入还缺以下关键配置：
  - `APP_BASE_URL`
  - `WECOM_CALLBACK_TOKEN`
  - `WECOM_CALLBACK_AES_KEY`
  - `VAULT_BASE_URL`
  - `VAULT_SHARED_TOKEN`
  - `VAULT_SECRET`
- 企业微信后台后续需要配置的内容已明确：
  - 自建应用主页：建议指向 `/community`
  - 接收消息回调 URL：`/answer/api/v1/wecom/callback`
  - Token：与 `WECOM_CALLBACK_TOKEN` 保持一致
  - EncodingAESKey：与 `WECOM_CALLBACK_AES_KEY` 保持一致
  - 网页授权及 JS-SDK 域名：正式部署域名
  - 可信域名：正式部署域名
  - 应用可见范围：按部门/成员配置
- 已确认的代码约束：
  - 企业微信 OAuth 回调地址由 `APP_BASE_URL + /answer/api/v1/wecom/auth/callback` 拼接生成
  - 主应用登录链依赖 Vault 服务完成 `corp_id + user_id -> anon_subject_id` 映射
- 当前前端社区入口仍是本地 `preview login`，尚未切换到 `/answer/api/v1/wecom/auth/start`

### 10. 预发运维拓扑与管理入口已确认

- 预发 ingress 目录：
  - `/opt/niming-community-ingress`
- 当前入口配置：
  - `Caddyfile` 将 `forum.xingyuanjituan.cn` 反代到 `127.0.0.1:9080`
  - `cloudflared` 负责公网入口隧道
- 后台入口说明已确认：
  - 用户登录页：`/users/login`
  - 管理后台入口：`/admin`
  - `/admin/login` 是后台“登录设置”页面，不是独立认证入口
- 数据库管理方式已确认：
  - 当前未部署独立 `pgAdmin` / `Adminer`
  - 现阶段通过 SSH + `docker exec` + `psql` 管理主库和 vault 库

## 当前未完成项

### 1. 企业微信回调已补到“可验证回调”阶段，事件分发仍未完成

- `internal/service/wecom/service.go` 中：
  - 已实现 callback 签名校验
  - 已实现 AES 解密
  - 已实现回调消息 XML 基础解析
  - 尚未实现消息卡片交互/业务事件分发
- 当前前端登录入口也尚未切换：
  - 社区页仍调用本地预览登录接口
  - 还未改为跳转企业微信授权入口

### 2. Vault 仍是最小可运行版本

- 还未补：
  - 更细粒度权限控制
  - Token 轮换策略
  - 更完整的错误码与管理接口
  - 生产级密钥管理

### 3. 企业微信/Vault 还未与本地预览前端做真正联调

- 当前本地预览模式完全绕过企业微信。
- 后续要进入企业微信实装阶段，还需要继续补：
  - 企微授权回跳到社区 H5
  - 企业微信用户与匿名档案映射联调
  - Vault reveal 审计流真实打通
  - 企业微信消息通知与卡片跳转

## 下一步优先级

1. 先让用户确认 `/community` 页面样式、频道结构和匿名展示是否符合预期
2. 收齐企业微信正式接入所缺配置：
   - `APP_BASE_URL`
   - `WECOM_CALLBACK_TOKEN`
   - `WECOM_CALLBACK_AES_KEY`
   - `VAULT_BASE_URL`
   - `VAULT_SHARED_TOKEN`
   - `VAULT_SECRET`
3. 继续补企业微信授权接入，替换本地 preview login
4. 跑通 Vault 与主应用联调，补 reveal 审计约束
5. 处理企业微信事件分发、应用消息、卡片回跳
6. 明确是否开放附件 / 视频上传，并补对应白名单与前端交互
7. 为 `/data/uploads` 与 PostgreSQL 主库补宿主机级备份方案

## 当前风险

- 当前匿名社区已经不是骨架演示，而是“本地可访问、可登录、可浏览、可发帖回帖”的预览状态。
- 本地 Docker 预览链已经固化，但完整重建时间仍偏长；这是 Apache Answer 主应用、前端静态资源和插件二次打包叠加造成的。
- 讨论频道仍然复用了 Answer/Comment 语义，虽然对预览足够，但正式上线前需要再评估是否满足产品语义。
- 企业微信部分仍属于接入骨架，不是生产可直接上线状态。
- 本地预览模式下的匿名身份是演示账号，不代表最终企业微信实名映射逻辑已经联通。
- 当前仍缺正式可公网访问的 `APP_BASE_URL` 与企业微信回调配置，因此无法直接完成企业微信后台联调。
- 当前是 `WSL2` 环境，本地局域网访问还依赖 Windows 侧端口转发，不适合作为企业微信正式回调地址。
- 预发环境虽然已上线，但当前附件扩展名白名单为空，视频/普通附件能力仍未真正开放。
- 当前非结构化文件仍落单机本地卷，尚未建立独立对象存储或跨机容灾。

## 本次继续开发的关键文件

- `internal/entity/question_entity.go`
- `internal/entity/anonymous_profile_entity.go`
- `internal/entity/moderation_action_entity.go`
- `internal/migrations/v32.go`
- `internal/schema/question_schema.go`
- `internal/schema/community_schema.go`
- `internal/schema/wecom_schema.go`
- `internal/service/content/question_service.go`
- `internal/service/question_common/question.go`
- `internal/service/community/service.go`
- `internal/service/community/preview.go`
- `internal/service/wecom/service.go`
- `internal/service/wecom/callback_service.go`
- `internal/controller/community_controller.go`
- `internal/controller/wecom_controller.go`
- `internal/router/answer_api_router.go`
- `internal/controller/permission_controller.go`
- `internal/base/constant/privilege.go`
- `internal/service/rank/rank_service.go`
- `internal/service/rank/rank_service_test.go`
- `internal/migrations/v34.go`
- `internal/migrations/v34_test.go`
- `deploy/docker-compose.community.predeploy.yml`
- `script/deploy_community_predeploy.sh`
- `internal/service/provider.go`
- `internal/controller/controller.go`
- `cmd/wire_gen.go`
- `internal/vaultapp/server.go`
- `cmd/vault/main.go`
- `deploy/docker-compose.community.yml`
- `deploy/docker-compose.community.preview.yml`
- `Dockerfile.vault`
- `Dockerfile`
- `ui/src/pages/Community/*`
- `ui/src/services/client/community.ts`
- `ui/src/router/routes.ts`

当前可视为：**匿名社区本地预览已经跑通，下一步重点转向页面确认、企业微信授权接入和 Vault 联调。**
