# Bug 是什么

模板复用会带回新模板已经删除的检查项，并让新旧发布版本共享证据切片。

# 如何触发

旧版本含“保留”和“已删除”两个答案，新模板只保留前者；创建复用版本后修改保留项证据，再读取旧版本。也可运行 `go test ./tests -run TestTemplateReuseFiltersRemovedItemsAndDoesNotAliasEvidence -count=1`。

# 错误信息

复现测试首先报告 `removed template item was reused`；绕过该断言后可观察旧版本证据被新版本修改。
