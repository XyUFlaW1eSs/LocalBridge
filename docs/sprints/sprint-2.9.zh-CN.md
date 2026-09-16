# Sprint 2.9：Windows 受保护凭据存储

[English](sprint-2.9.md)

## 结果

Windows 部署可用当前用户 DPAPI 保护管理 Bearer Token、配对码和 peer Token 轮换状态。配置引用在模块构造前解析；
注册表 v3 明文会原子迁移到受保护 v4；任意保护/解密错误都会使启动失败，且不改写来源文件。

## 已交付

- 可注入 `Protector`、Windows DPAPI 实现和明确不可用的非 Windows provider。
- 严格版本化、上限 1 MiB 的凭据存储：校验名称/值边界，拒绝重复/未知内容，以仅所有者临时文件刷盘并原子替换。
- `auto`、`required`、`disabled` 策略；管理/配对值引用；内联/引用冲突校验和解析后管理 Token 长度校验。
- `-credential-action set|delete|status` 与 `-credential-name`；隐藏终端或 stdin 输入，无秘密参数/输出，明确退出码，不启动服务。
- 注册表 v4 受保护的当前/上一代 Token、purpose 绑定、重启解密、v3 迁移、v4 `disabled` 到受保护模式的原子升级、磁盘明文排除，
  以及受保护到 disabled 或 provider 不匹配时保持原字节的关闭式拒绝。
- 脱敏配置/支持包保护元数据，不含秘密值、引用名、凭据存储路径或状态正文。
- 覆盖 DPAPI/存储 round trip、purpose 不匹配、损坏、大小/版本/重复、失败不覆盖、YAML/JSON 策略、应用解析、CLI、
  注册表迁移/轮换/重启和支持包脱敏的单元测试。

## 兼容性与限制

- Schema 保持 v1，因为新字段为增量字段，默认值保留 Windows 旧配置语义。来源 YAML 绝不自动改写。
- Windows `auto` 使用 DPAPI；`required` 额外拒绝内联秘密；显式 `disabled` 保留明文兼容，并在诊断中明确显示。
- 非 Windows 不支持生产保护；尚未实现 macOS Keychain 与 Linux Secret Service。DPAPI 文件不能跨用户或主机移植。
- 本 Sprint 不删除旧 YAML 内联值、不实现证书信任/轮换 UX，也不增加云端密钥服务。

## 验证契约

运行 `gofmt`、`go test ./...`、`go vet ./...`、`git diff --check`、Windows 构建/测试和
`GOOS=linux CGO_ENABLED=0 go build ./...`。迁移失败测试必须逐字节确认原文件不变；磁盘断言必须同时搜索当前与上一代明文 Token。
