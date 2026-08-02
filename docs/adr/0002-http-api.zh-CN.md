# ADR 0002：标准库 HTTP 与 JSON

[English version](0002-http-api.md)

## 决策

使用 `net/http`、Go 的支持方法匹配的 `ServeMux`，并在 `/api/v1` 下使用 JSON 负载。

## 理由

协议规模小，可以直接通过浏览器或快捷指令调试，同时避免引入框架依赖。版本化路径为未来客户端提供稳定的
兼容边界。
