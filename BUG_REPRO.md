# Bug 是什么

已过期或缺审批时间的豁免会让仍开放的阻塞项被门禁忽略，发布版本错误进入可发布状态。

# 如何触发

创建待复核版本及开放阻塞项，附加一个审批过但已到期的豁免，完成所需会签后执行门禁评估；或运行 `go test ./tests -run TestExpiredWaiverStillBlocksReleaseAcrossService -count=1`。

# 错误信息

复现测试报告 `expected open blocker, got state=ready err=<nil>`。
