# LEC 身份与代码组织接入

设置 `LEC_SSO_ENABLED=true` 后，控制台只展示 LecSSO 单点登录。登录链路校验 HTTPS、签名、issuer、audience、state、nonce 与 PKCE，再核实账号未停用且仍属于工作室。新成员首次登录创建零权限账号，由管理员选择成员并按应用、环境授权；原生管理员和 API Token 保留用于运维。

## 服务端配置

以下值放在部署 Secret 中，不提交到 Git，也不返回控制台。

| 配置 | 含义 |
| --- | --- |
| `LEC_PUBLIC_ORIGIN` | 平台 HTTPS origin，不含路径 |
| `LEC_SSO_ISSUER` | LecSSO OIDC issuer |
| `LEC_SSO_CLIENT_ID` / `LEC_SSO_CLIENT_SECRET` | Traditional 登录客户端 |
| `LEC_SSO_MANAGEMENT_ENDPOINT` / `LEC_SSO_MANAGEMENT_RESOURCE` | 成员查询 API 和 resource |
| `LEC_SSO_MANAGEMENT_CLIENT_ID` / `LEC_SSO_MANAGEMENT_CLIENT_SECRET` | 独立的成员查询 M2M 客户端 |
| `LEC_SSO_ORGANIZATION_ID` | 正式成员所属组织 |
| `LEC_GITHUB_ORG` | 固定为 `lec-org` |
| `LEC_GITHUB_APP_ID` / `LEC_GITHUB_INSTALLATION_ID` | 组织 App 与安装 ID |
| `LEC_GITHUB_PRIVATE_KEY_FILE` | 只读挂载的 RSA 私钥文件 |
| `SSL_CERT_FILE` | 系统 CA 与开发证书组成的信任包，仅开发环境需要额外证书 |

登录回调为 `${LEC_PUBLIC_ORIGIN}/orchestrator/auth/callback`。回跳仅允许同源 `/dashboard`。成员资格成功查询缓存 15 秒；查询失败不延长缓存。当前 Logto OSS Management API 只接受 `all` scope：这个独立 M2M 凭据拥有管理能力，代码仅调用成员查询 GET，必须只保存在服务端，不能把它称为只读权限凭据。

## GitHub 与镜像权限

组织拥有的 GitHub App 仅申请 Contents 和 Metadata 只读，安装到组织全部仓库。`pkg/lec` 校验 App、安装和组织对应关系。仓库目录只向平台管理员开放；手填或变量解析后的仓库地址仍由服务端校验组织。

系统账户 `lec-org GitHub App` 每 5 分钟同步临时凭据给 Git Sensor；原生轮询可在没有公网 webhook 的内网运行。实际 CI 触发时重新签发限定到该仓库的一小时 Token，不把全组织 Token 交给构建脚本。管理员应给普通成员分配查看或触发权限，仓库绑定和流水线配置由管理员维护。

GHCR 目前使用单独的 `write:packages` PAT。它只负责推镜像，不负责读取源码；组织 App 与 PAT 是两套独立凭据。PAT 到期前需要在镜像仓库凭据表单轮换，后续可以改用团队机器人账号或团队自建镜像仓库。

GitHub App Token 长于旧 PAT。部署本分支须执行 `37704800_lec_github_app_token.up.sql`，并把 Git Sensor 数据库的 `git_provider.access_token` 同样扩展为 `text`。先备份两个数据库。回退不能截断长凭据；down 迁移会拒绝不安全的回退。

## 新增 API

| 方法和路径 | 作用 | 权限 |
| --- | --- | --- |
| `GET /orchestrator/user/lec/members` | 选择可授权的正式成员，只返回稳定账号键、姓名与 Handle | 平台管理员 |
| `GET /orchestrator/git/lec/repositories` | 选择组织 App 授权的仓库，不返回 Token | 平台管理员 |

原生 API 响应仍采用 `code`、`result`、`errors`。认证回调、Git 账户和镜像仓库操作仅记录审计元数据，不记录请求正文及回调查询参数。

## 验证

```sh
go test -race ./pkg/lec
go test ./api/util ./pkg/build/git/gitProvider ./pkg/auth/authentication ./pkg/auth/sso ./pkg/auth/authorisation/casbin
go build -mod=vendor .
```

OIDC 测试使用真实签名及 HTTPS 测试服务器，覆盖错误 issuer、audience、签名、nonce、state、外部回跳、停用与非成员等拒绝路径。GitHub App 测试校验 App 签名、组织、只读权限、短期凭据缓存与单仓库范围。完整上游测试仍有既存编译和参考 Chart 路径问题，不能据此宣称全量测试通过。
