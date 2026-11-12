# 接口中文消息

`zh-CN.json` 保存系统消息的中文译文，使用 `go:embed` 随服务编译，不依赖外部翻译服务。

两个 `ApiError.MarshalJSON` 只在响应序列化时翻译 `userMessage` 和 `userDetailMessage`。原始 `internalMessage`、错误码、`Error()` 和业务判断保持不变。未知第三方错误使用中文提示，原始诊断仍可用于排障。成功响应只替换词典中的已知消息，未知令牌和资源标识符保持原样。

统一响应还会翻译结果对象中已登记的 `message` 字段。JSON 数字使用 `json.Number` 保留精度，名称、描述、令牌等数据字段不参与翻译。

新增消息应登记原文和中文译文。格式化消息必须保留占位符的类型、数量和语义顺序：例如 `%s`、`%d`、`%v`。译文需要调整语序时，应以先说明参数、再补充上下文的方式保持原参数顺序。测试覆盖参数保留、未知错误回退和错误序列化不改变内部行为。

运行：

```sh
go test ./internal/i18n ./internal/util ./api/restHandler/common -run 'TestTranslate|TestUnknownErrorFallback|TestResponseMessagesPreserveData|TestApiErrorChineseWireMessage|TestChineseResponseContract'
```

接口协议字段、Kubernetes 原始资源、命令、日志和用户输入不属于界面文案。
