# Bug 是什么

已发布快照与查询返回值共享内存，且填报接口仍允许修改发布版本，历史放行依据不再只读。

# 如何触发

发布一个带答案的版本，读取后直接修改返回快照，再次查询；随后对同一版本调用填报。也可运行 `go test ./tests -run TestReleasedSnapshotCannotBeMutatedThroughReadsOrAnswers -count=1`。

# 错误信息

复现测试首先报告 `read alias mutated stored snapshot: "tampered"`。
