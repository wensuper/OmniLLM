# 更新记录

## v0.1.0-dev（2026-10-18）

- 库：`hub`（主机）、`device`（设备）、`relay`（连接服务）、`localapi`（本机 OpenAI 兼容接口）。
- 端到端加密（Noise IK / IKpsk2），连接服务看不到内容。
- 唯一上游出口：限流、退避、熔断、人优先；预算按最坏情况预留。
- 配对：配对文字 + 主机上确认 6 位校验码，配对后更换信箱口令。
- 命令行 `omnillm`：主机、设备、连接服务。
- 测试：单元、模糊、竞态、端到端（抓取连接服务流量）、对抗、两轮独立审查的回归、静态分析、真实程序冒烟。
- 本机控制台和首次运行向导；二维码配对；`omnillm install / uninstall / status / open`；写给 AI 的 INSTALL.md 和校验下载的 install.sh；GoReleaser 签名发布和 CI。
- 还没做的见 docs/design.md §13.3。
