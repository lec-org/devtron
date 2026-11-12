说明：本文介绍 Devtron 的上游能力。需要了解 CI/CD，请直接阅读[Devtron 平台](#devtron-platform)。

本仓库由 `lec-org` 维护，上游为 [devtron-labs/devtron](https://github.com/devtron-labs/devtron)。当前分支已加入中文界面与接口消息支持，维护方式见[接口中文消息](internal/i18n/README.md)。下面的上游 Helm 安装命令不会自动安装本分支的中文镜像；LEC 的部署与 SSO 接入尚待实施。
#
<img referrerpolicy="no-referrer-when-downgrade" src="https://static.scarf.sh/a.png?x-pxid=40318be7-e780-4186-8959-0078e057dddc" />
<p align="center">
<picture>
  <source media="(prefers-color-scheme: dark)"  srcset="./assets/devtron-darkmode-logo.png">
  <source media="(prefers-color-scheme: light)"  srcset="./assets/devtron-lightmode-logo.png">
  <img width="333.333" height="260" src="./assets/devtron-logo-dark-light.png">
</picture>
<h1 align= "center">AI 原生 Kubernetes 管理平台</h1>
<p align= "center">Kubernetes 功能强大，但管理复杂。Devtron 将应用和基础设施管理集中到同一平台，并通过 AI 助手简化运维、加快交付。</p>
</p>

<p align="center">
<br>
<a href="https://docs.devtron.ai/" rel="nofollow"><strong>查看文档</strong></a>
·
<a href="https://license.devtron.ai/dashboard" rel="nofollow"><strong>体验 Devtron SaaS</strong></a>
<br>
<br>
<a href="https://devtron.ai/">官方网站</a>
·
<a href="https://devtron.ai/blog/">博客</a>
·
<a href="https://discord.gg/jsRG5qx2gp">加入 Discord 社区</a>
.
<a href="https://www.youtube.com/channel/UCAHRp9qp0z1y9MMtQlcFtcw">YouTube</a>


</p>
<p align="center">
<a href="https://discord.gg/jsRG5qx2gp"><img src="https://img.shields.io/discord/687207715902193673?logo=discord&label=Discord&color=5865F2&logoColor=white" alt="Join Discord"></a>
<a href="https://goreportcard.com/badge/github.com/devtron-labs/devtron"><img src="https://goreportcard.com/badge/github.com/devtron-labs/devtron" alt="Go Report Card"></a>
<a href="./LICENSE"><img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="License"></a>
<a href="./CONTRIBUTING.md"><img src="https://img.shields.io/github/contributors/devtron-labs/devtron" alt="contributors"></a>
<a href="https://github.com/devtron-labs/devtron"><img src="https://img.shields.io/github/commit-activity/m/devtron-labs/devtron" alt="Commit Activity"></a>
<a href="https://bestpractices.coreinfrastructure.org/projects/4411"><img src="https://bestpractices.coreinfrastructure.org/projects/4411/badge" alt="CII Best Practices"></a>
<a href="http://golang.org"><img src="https://img.shields.io/badge/Made%20with-Go-1f425f.svg" alt="made-with-Go"></a>
<a href="http://devtron.ai/"><img src="https://img.shields.io/website-up-down-green-red/http/shields.io.svg" alt="Website devtron.ai"></a>
<a href="https://github.com/devtron-labs/devtron/releases"><img src="https://img.shields.io/github/v/release/devtron-labs/devtron"></a>
<a href="https://twitter.com/intent/tweet?text=Devtron%20helps%20in%20simplifying%20software delivery%20workflow%20for%20Kubernetes,%20check%20it%20out!!%20&hashtags=OpenSource,Kubernetes,DevOps,CICD,go&url=https://github.com/devtron-labs/devtron%0a"><img src="https://img.shields.io/twitter/url/http/shields.io.svg?style=social" alt="Tweet"></a>
<h1></h1>

Devtron 的**可扩展 Kubernetes 控制台**通过统一界面展示集群状态，简化 Helm 应用管理。<br>
内置 RBAC 用于控制访问权限，并集中展示通过 **ArgoCD**、**FluxCD** 等 GitOps 工具部署在不同集群中的工作负载。上游将其定位为统一的 DevOps 工作台，并宣称运维效率最高可提升 20 倍；这不是 LEC 环境的实测结果。

通过下面的视频了解 **Kubernetes 控制台**的功能。

<a href="https://youtu.be/oqCAB9b-SGQ?si=YoUJfHL43VXRU5wx">
<br>
<p align="center"><img src="./assets/dashboard.png"></p>
</a>

Devtron 的 Kubernetes 控制台提供以下基础能力：
- [Helm 应用管理](https://docs.dashboard.devtron.ai/user-guide/helm-apps) 用于部署、配置和管理 Helm 应用
- [资源浏览器](https://docs.dashboard.devtron.ai/user-guide/resource-browser) 用于查看和管理 Node、Pod、ConfigMap、自定义资源定义（CRD）等集群资源
- [单点登录（SSO）](https://docs.dashboard.devtron.ai/operator-guide/authorization/sso-login-services) 用于简化团队成员的接入与认证。
- [细粒度 RBAC](https://docs.dashboard.devtron.ai/operator-guide/authorization) 用于控制用户对不同控制台功能和集群资源的访问权限。

[Devtron](#devtron-platform) 可用于部署、观察、管理和调试各集群中已有的 Helm 应用。

---

## Devtron 控制台

Devtron 的 Kubernetes 控制台提供多种集群管理能力，可按团队需要选用。

### 功能

- 简化 Helm 应用的配置和管理，支持版本回退
- 比较并处理不同环境之间的配置差异
- 管理多个 Kubernetes 环境中的 Helm 部署
- 在同一界面查看多个集群中的 Helm、ArgoCD 和 FluxCD 应用
- 集中查看和管理多个集群中的 K8s 资源
- 通过 SSO 与 RBAC 管理团队成员的登录和访问权限

完整功能介绍见[控制台文档](https://docs.dashboard.devtron.ai/)。

### 安装

开始前，请准备 [Kubernetes 集群](https://kubernetes.io/docs/tutorials/kubernetes-basics/create-cluster/)并安装 [Helm](https://helm.sh/docs/intro/install/)。兼容版本请以当前官方安装文档为准。

### 安装 Devtron 控制台

使用以下上游命令安装 Devtron 控制台：

```bash
helm repo add devtron https://helm.devtron.ai

helm install devtron devtron/devtron-operator \
--create-namespace --namespace devtroncd
```

### 访问 Devtron

**访问地址**：使用以下命令获取控制台地址：

```bash
kubectl get svc -n devtroncd devtron-service -o jsonpath='{.status.loadBalancer.ingress}'
```

**登录凭据**：

**用户名**：  `admin` <br>
**密码**：Devtron v0.6.0 及以上版本使用以下命令获取管理员密码：

```bash
kubectl -n devtroncd get secret devtron-secret -o jsonpath='{.data.ADMIN_PASSWORD}' | base64 -d
```

早于 v0.6.0 的 Devtron 版本使用以下命令获取管理员密码：

```bash
kubectl -n devtroncd get secret devtron-secret -o jsonpath='{.data.ACD_PASSWORD}' | base64 -d
```

更多说明见[访问 Devtron 控制台](https://docs.dashboard.devtron.ai/getting-started/install-devtron-dashboard)。

---

<a id="devtron-platform"></a>

## Devtron 平台

Devtron 平台将 DevOps 相关能力集中到同一工作台，用于管理多个 Kubernetes 集群中的应用。<br>

上游平台还介绍了金丝雀发布、A/B 测试、自动回退等高级能力，用于简化应用交付和 Kubernetes 运维。这些描述不代表相关功能已在 LEC 部署或启用；具体能力还需核对所使用的版本和许可。

<p align="center"><img src="./assets/devtron-hero-image.webp"></p>

Devtron 通过统一界面集成微服务生命周期中的 CI、CD、安全、成本、调试和可观测性工具，帮助团队管理各 Kubernetes 集群中的应用。

### Devtron 集成

通过不同集成，Devtron 控制台可覆盖从 CI/CD 流水线到日常维护的 Kubernetes 应用工作流。

Devtron 提供以下集成：

- [构建与部署（CI/CD）](https://docs.devtron.ai/usage/integrations/build-and-deploy-ci-cd) 用于自动构建和部署应用，让团队专注于业务需求、代码质量与安全
- [GitOps（Argo CD）](https://docs.devtron.ai/usage/integrations/argocd) 通过 ArgoCD 实现 GitOps 持续交付
- [漏洞扫描（Clair）](https://docs.devtron.ai/usage/integrations/clair) 用于在镜像构建阶段扫描漏洞。
- [漏洞扫描（Trivy）](https://docs.devtron.ai/usage/integrations/clair) 用于在镜像构建阶段扫描漏洞。
- [通知](https://docs.devtron.ai/usage/integrations/notifications) 通过 SES 或 SMTP，将流水线触发、成功和失败事件发送到邮箱或 Slack 频道。
- [监控（Grafana）](https://docs.devtron.ai/usage/integrations/grafana) 通过 Grafana 展示 CPU、内存使用率、HTTP 状态码、吞吐量和延迟等应用指标。


### 功能

- 通过 Helm 或 ArgoCD GitOps 提供端到端 CI/CD 流水线，支持可视化配置交付流程
- 通过一套 Devtron 向云端或自建的多个 Kubernetes 集群部署应用
- 通过 Trivy、Clair 等工具接入 DevSecOps 安全检查
- 在统一控制台中处理应用调试
- 展示 MTTR、变更失败率、部署频率和部署规模等交付指标

完整功能介绍见[平台文档](https://docs.devtron.ai/)。

### 安装 Devtron 平台

使用以下上游命令安装 Devtron 平台：

```bash
helm install devtron devtron/devtron-operator --namespace devtroncd \
--set installer.modules={cicd} --set argo-cd.enabled=true \
--set security.enabled=true  --set notifier.enabled=true  \
--set security.trivy.enabled=true --set monitoring.grafana.enabled=true
```

---

## 核心工具与技术

Devtron 的架构包含以下组件：

| 图标 | 工具 | 用途 | GitHub 仓库 | 许可证 |
| -------- | -------- | ---------- | ------------------- | ------- |
| <img src="./assets/readme-logos/kubernetes.jpg" width="64px"> | **Kubernetes** | 容器编排平台，管理部署、扩缩容和容器生命周期。                                                                         | [kubernetes/kubernetes](https://github.com/kubernetes/kubernetes) | [Apache License 2.0](https://github.com/kubernetes/kubernetes/blob/master/LICENSE) |
| <img src="./assets/readme-logos/golang.jpg" width="64px">     | **Golang**     | Devtron 核心服务使用的后端语言。                                                                           | [golang/go](https://github.com/golang/go)                         | [BSD 3-Clause License](https://github.com/golang/go/blob/master/LICENSE)           |
| <img src="./assets/readme-logos/helm.jpg" width="64px">       | **Helm**       | 管理 Kubernetes 应用打包、版本和部署流程。                                                                                           | [helm/helm](https://github.com/helm/helm)                         | [Apache License 2.0](https://github.com/helm/helm/blob/main/LICENSE)               |
| <img src="./assets/readme-logos/trivy.jpg" width="64px">      | **Trivy**      | 扫描漏洞、错误配置、敏感信息及 SBOM。 | [aquasecurity/trivy](https://github.com/aquasecurity/trivy)       | [Apache License 2.0](https://github.com/aquasecurity/trivy/blob/main/LICENSE)      |
| <img src="./assets/readme-logos/nats.jpg" width="64px">       | **NATS**       | 为 Devtron 微服务和工作流提供可靠消息传输。                                                     | [nats-io/nats-server](https://github.com/nats-io/nats-server)     | [Apache License 2.0](https://github.com/nats-io/nats-server/blob/main/LICENSE)     |
| <img src="./assets/readme-logos/dex.jpg" width="64px">        | **Dex**        | 对接身份提供方，提供 SSO 认证。                                                                       | [dexidp/dex](https://github.com/dexidp/dex)                       | [Apache License 2.0](https://github.com/dexidp/dex/blob/master/LICENSE)              |
| <img src="./assets/readme-logos/casbin.jpg" width="64px">     | **Casbin**     | 执行细粒度 RBAC 授权策略。                                                                             | [casbin/casbin](https://github.com/casbin/casbin)                 | [Apache License 2.0](https://github.com/casbin/casbin/blob/master/LICENSE)         |


---

## 可选组件

Devtron 支持以下可选组件：

| 图标 | 工具 | 用途 | GitHub 仓库 | 许可证 |
| -------- | -------- | ---------- | ------------------- | ------- |
| <img src="./assets/readme-logos/argo.jpg" width="64px">                 | **Argo CD / Argo Workflows**       | 提供 GitOps 部署和工作流自动化          | [argoproj/argo-cd](https://github.com/argoproj/argo-cd)                   | [Apache License 2.0](https://github.com/argoproj/argo-cd/blob/master/LICENSE)       |
| <img src="./assets/readme-logos/clair.jpg" width="64px">                | **Clair**                          | 扫描容器镜像中的已知漏洞                    | [quay/clair](https://github.com/quay/clair)                               | [Apache License 2.0](https://github.com/quay/clair/blob/main/LICENSE)                 |
| <img src="./assets/readme-logos/keda.jpg" width="64px">                 | **KEDA**                           | 为 Kubernetes 应用提供事件驱动的自动扩缩容                       | [kedacore/keda](https://github.com/kedacore/keda)                         | [Apache License 2.0](https://github.com/kedacore/keda/blob/main/LICENSE)            |
| <img src="./assets/readme-logos/k8s_external_secrets.jpg" width="64px"> | **Kubernetes External Secrets**    | 将外部密钥安全地同步到 Kubernetes 集群                           | [external-secrets/external-secrets](https://github.com/external-secrets/external-secrets) | [Apache License 2.0](https://github.com/external-secrets/external-secrets/blob/main/LICENSE) |
| <img src="./assets/readme-logos/prometheus.jpg" width="64px">           | **Prometheus**                     | 采集指标，用于监控、告警和性能分析               | [prometheus/prometheus](https://github.com/prometheus/prometheus)         | [Apache License 2.0](https://github.com/prometheus/prometheus/blob/main/LICENSE)      |

---

## :question: 常见问题与故障排查

- 排查 Devtron 故障时，请[参阅此文档](https://docs.devtron.ai/resources/devtron-troubleshoot)

## :busts_in_silhouette: 社区

了解 Devtron 的开发进展，与项目维护者、贡献者和社区成员交流：
- 关注 [Twitter 上的 @DevtronL](https://twitter.com/DevtronL)
- 通过 [GitHub Issues] 提出功能建议、改进意见或报告缺陷(https://github.com/devtron-labs/devtron/issues)
- 文章、操作指南和教程见 [Devtron 博客](https://devtron.ai/blog/)
- Devtron 在多个社区中得到使用，组织名单见[使用者清单](./USERS.md)。

### 加入 Discord 社区
<p>
<a href="https://discord.gg/jsRG5qx2gp">
   <img
   src="https://invidget.switchblade.xyz/jsRG5qx2gp"
   alt="Join Devtron : Heroku for Kubernetes"
   >
</a>
</p>

---

## :handshake: 参与贡献

请先阅读[贡献指南](CONTRIBUTING.md)，了解问题反馈方式、代码规范和开发流程。感谢你的贡献。

欢迎查看[社区贡献](COMMUNITY_CONTRIBUTIONS.md)，也欢迎制作 Devtron 视频或文章并加入清单。

### 贡献者

感谢所有贡献者的参与！

<a href="https://github.com/devtron-labs/devtron/graphs/contributors">
 <img src="https://contrib.rocks/image?repo=devtron-labs/devtron" />
</a>

---

## :bug: 漏洞报告

Devtron 重视安全和用户信任。如果发现安全问题，请发送到 <b>security@devtron.ai</b>。

---

## :bookmark: 许可证

Devtron 使用 [Apache License 2.0](LICENSE) 许可证。
