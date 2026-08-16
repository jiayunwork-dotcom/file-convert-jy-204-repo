// Package convert 在 CSV 与 JSON 之间批量转换文件，并为每个转换产出
// 一份元数据包（来源/目标、格式、行数、字节数、SHA-256、转换时间）。
// 仅依赖标准库，离线可构建。
package convert

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Meta 描述一次文件转换及其元数据。
type Meta struct {
	Source      string `json:"source"`
	Dest        string `json:"dest"`
	From        string `json:"from"`
	To          string `json:"to"`
	Rows        int    `json:"rows"`
	BytesIn     int64  `json:"bytes_in"`
	BytesOut    int64  `json:"bytes_out"`
	SHA256In    string `json:"sha256_in"`
	SHA256Out   string `json:"sha256_out"`
	ConvertedAt string `json:"converted_at"`
}

func readAll(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return fmt.Sprintf("%x", sum)
}

// ConvertFile 将单个文件从 from 格式转为 to 格式，写入 dst，返回元数据。
// 仅支持 csv<->json。
func ConvertFile(src, dst, from, to string) (Meta, error) {
	in, err := readAll(src)
	if err != nil {
		return Meta{}, err
	}

	var out []byte
	var rows int
	switch {
	case from == "csv" && to == "json":
		recs, err := csvToRows(in)
		if err != nil {
			return Meta{}, err
		}
		rows = len(recs)
		out, err = json.MarshalIndent(recs, "", "  ")
		if err != nil {
			return Meta{}, err
		}
	case from == "json" && to == "csv":
		recs, err := jsonToRows(in)
		if err != nil {
			return Meta{}, err
		}
		rows = len(recs)
		out, err = rowsToCSV(recs)
		if err != nil {
			return Meta{}, err
		}
	default:
		return Meta{}, fmt.Errorf("不支持的转换: %s -> %s（仅支持 csv<->json）", from, to)
	}

	if err := os.WriteFile(dst, out, 0o644); err != nil {
		return Meta{}, err
	}

	return Meta{
		Source:      src,
		Dest:        dst,
		From:        from,
		To:          to,
		Rows:        rows,
		BytesIn:     int64(len(in)),
		BytesOut:    int64(len(out)),
		SHA256In:    sha256Hex(in),
		SHA256Out:   sha256Hex(out),
		ConvertedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func csvToRows(b []byte) ([]map[string]string, error) {
	b = bytes.TrimPrefix(b, []byte("\xef\xbb\xbf")) // 去除 UTF-8 BOM
	r := csv.NewReader(bytes.NewReader(b))
	r.FieldsPerRecord = -1
	r.TrimLeadingSpace = true
	recs, err := r.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV 失败: %w", err)
	}
	if len(recs) == 0 {
		return nil, fmt.Errorf("CSV 为空")
	}
	header := recs[0]
	rows := make([]map[string]string, 0, len(recs)-1)
	for _, rec := range recs[1:] {
		row := make(map[string]string, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			} else {
				row[h] = ""
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func jsonToRows(b []byte) ([]map[string]string, error) {
	var arr []map[string]interface{}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	if err := dec.Decode(&arr); err != nil {
		// 允许单个对象（自动包装为单行）
		var one map[string]interface{}
		if err2 := json.Unmarshal(b, &one); err2 != nil {
			return nil, fmt.Errorf("解析 JSON 失败: %w", err)
		}
		arr = []map[string]interface{}{one}
	}
	rows := make([]map[string]string, 0, len(arr))
	for _, m := range arr {
		row := make(map[string]string, len(m))
		for k, v := range m {
			row[k] = valueToString(v)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// valueToString 将任意 JSON 值转为用于 CSV 单元格的字符串。
func valueToString(v interface{}) string {
	switch x := v.(type) {
	case nil:
		return ""
	case string:
		return x
	case json.Number:
		return x.String()
	case bool:
		if x {
			return "true"
		}
		return "false"
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%v", x)
		}
		return string(b)
	}
}

func rowsToCSV(rows []map[string]string) ([]byte, error) {
	if len(rows) == 0 {
		return nil, fmt.Errorf("JSON 中无记录可写")
	}
	keySet := make(map[string]struct{}, len(rows)*4)
	for _, r := range rows {
		for k := range r {
			keySet[k] = struct{}{}
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Sort(sort.Reverse(sort.StringSlice(keys)))

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(keys); err != nil {
		return nil, err
	}
	for _, r := range rows {
		rec := make([]string, len(keys))
		for i, k := range keys {
			rec[i] = r[k]
		}
		if err := w.Write(rec); err != nil {
			return nil, err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// ExtFor 返回格式对应的文件扩展名（不含点）。
func ExtFor(format string) string {
	switch format {
	case "csv":
		return "csv"
	case "json":
		return "json"
	default:
		return format
	}
}

// MatchExt 判断文件名是否以给定格式扩展名结尾。
func MatchExt(name, format string) bool {
	return strings.EqualFold(filepath.Ext(name), "."+ExtFor(format))
}
