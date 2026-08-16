# file-convert 示例

目录下含两个样例文件，覆盖两种转换方向：

- `users.csv` → 转 JSON
- `products.json` → 转 CSV

## 运行

```bash
# CSV 目录 -> JSON（默认写入 example/converted/ 子目录，避免覆盖源文件）
go run . -src example -from csv -to json

# JSON 目录 -> CSV
go run . -src example -from json -to csv

# 指定独立输出目录，并关闭元数据
go run . -src example -from csv -to json -out /tmp/out -meta=false
```

## 期望行为

- 每个源文件在输出目录（默认 `<src>/converted/`）生成同名目标文件（`users.csv` → `users.json`，`products.json` → `products.csv`）。
- 同时写出 `users.json.meta.json` 等元数据：记录来源/目标、格式、行数、输入/输出字节数、SHA-256、转换时间（本工具会跳过自身生成的 `.meta.json`，避免递归转换）。
- 坏输入受控报错：缺 `-src/-from/-to` 返回用法错误（exit 2）；源目录不存在、无匹配文件、解析失败均返回 exit 1，不 panic。
