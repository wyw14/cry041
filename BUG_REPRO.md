# Bug 是什么

陈旧页面携带旧 revision 填报清单时不返回乐观锁冲突，答案被写入且 revision 异常跳号。

# 如何触发

让两个操作员都读取 revision 1；第一个保存“备份”答案后，第二个仍以 revision 1 保存“监控”答案。也可运行 `go test ./tests -run TestStaleChecklistWriterCannotMergeAgainstNewerRevision -count=1`。

# 错误信息

复现测试报告 `stale writer should conflict, got <nil>`，并可观察陈旧答案进入发布清单。
