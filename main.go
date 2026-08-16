// Command file-convert 批量将目录下的 CSV/JSON 文件在两种格式间互转，
// 并为每个转换写出一份 .meta.json 元数据（行数、字节数、SHA-256 等）。
// 纯标准库实现，离线可构建。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"file-convert/internal/convert"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("file-convert", flag.ContinueOnError)
	fs.SetOutput(stderr)
	src := fs.String("src", "", "源目录（必填）")
	from := fs.String("from", "", "源格式: csv|json（必填）")
	to := fs.String("to", "", "目标格式: csv|json（必填）")
	out := fs.String("out", "", "输出目录（默认与源同目录）")
	meta := fs.Bool("meta", true, "是否额外写出 .meta.json 元数据")
	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *src == "" || *from == "" || *to == "" {
		fmt.Fprintln(stderr, "错误: -src / -from / -to 均为必填参数")
		fmt.Fprintln(stderr, "用法: file-convert -src DIR -from csv -to json [-out DIR] [-meta=false]")
		return 2
	}
	if *from == *to {
		fmt.Fprintf(stderr, "错误: 源格式与目标格式不能相同（%s）\n", *from)
		return 2
	}
	if !isValidFormat(*from) || !isValidFormat(*to) {
		fmt.Fprintf(stderr, "错误: 格式必须是 csv 或 json（收到 from=%s to=%s）\n", *from, *to)
		return 2
	}

	entries, err := os.ReadDir(*src)
	if err != nil {
		fmt.Fprintf(stderr, "读取源目录失败 %s: %v\n", *src, err)
		return 1
	}

	outDir := *out
	if outDir == "" {
		// 默认写入源目录下的 converted/ 子目录，避免覆盖源文件或被二次扫描
		outDir = filepath.Join(*src, "converted")
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "创建输出目录失败 %s: %v\n", outDir, err)
		return 1
	}

	ext := convert.ExtFor(*from)
	converted := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		// 跳过本工具自己写出的元数据 sidecar，避免递归转换
		if strings.HasSuffix(e.Name(), ".meta.json") {
			continue
		}
		if !convert.MatchExt(e.Name(), *from) {
			continue
		}
		base := e.Name()[:len(e.Name())-len(filepath.Ext(e.Name()))]
		srcPath := filepath.Join(*src, e.Name())
		dstPath := filepath.Join(outDir, base+"."+convert.ExtFor(*to))

		m, err := convert.ConvertFile(srcPath, dstPath, *from, *to)
		if err != nil {
			fmt.Fprintf(stderr, "转换失败 %s: %v\n", e.Name(), err)
			return 1
		}
		converted++

		fmt.Fprintf(stdout, "已转换: %s -> %s (行数=%d, 字节 %d->%d)\n",
			e.Name(), filepath.Base(dstPath), m.Rows, m.BytesIn, m.BytesOut)

		if *meta {
			metaPath := dstPath + ".meta.json"
			mb, err := json.MarshalIndent(m, "", "  ")
			if err != nil {
				fmt.Fprintf(stderr, "写出元数据失败 %s: %v\n", e.Name(), err)
				return 1
			}
			if err := os.WriteFile(metaPath, mb, 0o644); err != nil {
				fmt.Fprintf(stderr, "写出元数据失败 %s: %v\n", e.Name(), err)
				return 1
			}
			fmt.Fprintf(stdout, "  元数据: %s\n", filepath.Base(metaPath))
		}
	}

	if converted == 0 {
		fmt.Fprintf(stderr, "警告: 在 %s 中未找到任何 .%s 文件\n", *src, ext)
		return 1
	}
	fmt.Fprintf(stdout, "完成: 共转换 %d 个文件\n", converted)
	return 0
}

func isValidFormat(f string) bool {
	return f == "csv" || f == "json"
}
