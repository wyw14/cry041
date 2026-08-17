# Bug 是什么

模拟发布请求超时后任务继续运行并显示成功，失败码、错误信息和恢复指引未保存在历史记录中。

# 如何触发

用 10ms 截止时间执行一个延迟 150ms 的本地发布适配器，再读取返回值和执行历史；或运行 `go test ./tests -run TestCancelledDeploymentIsFailedAndKeepsRecoveryEvidence -count=1`。

# 错误信息

复现测试报告 `failure evidence lost`，状态为 `succeeded` 且失败字段为空。
