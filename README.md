# save-shard

存档分片：分片大小校验、写入幂等、缺片检测与原子提交，只用 Go 标准库。

```
go run ./cmd/save-shard --sample=size
go run ./cmd/save-shard --sample=dup
go run ./cmd/save-shard --sample=missing
go run ./cmd/save-shard --sample=scan
go test ./...
go vet ./...
```

## 口径（README 为准）

- **分片大小**：除最后一片外，每个分片长度必须等于 `shardSize`；不合法返回 false 且不写入。
- **幂等**：同一个 `reqID` 只写入一次，重复调用返回 false。
- **缺片检测**：`Commit` 要求分片下标从 0 连续无缺，否则返回 `ErrIncomplete`。
- **原子提交**：只有校验通过才标记已提交；失败时不改变提交状态。
- **不变量**：已提交的存档不会因为再写入而改变；同一串写入与提交序列重复执行结果相同。
- **代价**：`Commit` 不许重扫全部存档，`Scanned` 不随存档数线性增长。
- **规模**：10 万个存档、每秒 10 万次提交，单次摊还 O(该存档的分片数)。

## 输出契约（不改格式）

```
ok=..
second=..
err=.. committed=..
scanned=..
```

## 目录

```
shard.go              大小、幂等、缺片与提交
shard_test.go         单元测试
cmd/save-shard/       命令行入口
```
