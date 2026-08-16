package convert

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestCSVToRows(t *testing.T) {
	csv := "name,age,city\nAlice,30,Beijing\nBob,25,Shanghai\n"
	rows, err := csvToRows([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("行数 = %d, want 2", len(rows))
	}
	if rows[0]["name"] != "Alice" || rows[0]["age"] != "30" || rows[0]["city"] != "Beijing" {
		t.Errorf("首行解析错误: %v", rows[0])
	}
	if rows[1]["name"] != "Bob" {
		t.Errorf("次行解析错误: %v", rows[1])
	}
}

func TestJSONToRows(t *testing.T) {
	jsonB := `[{"name":"Alice","age":"30"},{"name":"Bob","age":"25"}]`
	rows, err := jsonToRows([]byte(jsonB))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("行数 = %d, want 2", len(rows))
	}
	if rows[1]["name"] != "Bob" {
		t.Errorf("次行解析错误: %v", rows[1])
	}

	one := `{"k":"v"}`
	rows2, err := jsonToRows([]byte(one))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows2) != 1 || rows2[0]["k"] != "v" {
		t.Errorf("单对象解析错误: %v", rows2)
	}
}

func TestRowsToCSV(t *testing.T) {
	rows := []map[string]string{
		{"name": "Alice", "age": "30"},
		{"age": "25", "name": "Bob"}, // 乱序键，应稳定按字母排
	}
	out, err := rowsToCSV(rows)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	want := "age,name\n30,Alice\n25,Bob\n"
	if got != want {
		t.Errorf("rowsToCSV =\n%q\nwant\n%q", got, want)
	}
}

func TestRoundTripCSVJSONCSV(t *testing.T) {
	csv := "name,age\nAlice,30\nBob,25\n"
	rows, err := csvToRows([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	jsonB, err := json.MarshalIndent(rows, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	rows2, err := jsonToRows(jsonB)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(rows, rows2) {
		t.Errorf("往返不一致:\n%v\n%v", rows, rows2)
	}
}

func TestUnsupportedConversion(t *testing.T) {
	_, err := ConvertFile("x.csv", "y.txt", "csv", "txt")
	if err == nil {
		t.Error("不支持的格式组合应报错")
	}
}

func TestExtForAndMatch(t *testing.T) {
	if ExtFor("csv") != "csv" || ExtFor("json") != "json" {
		t.Error("ExtFor 错误")
	}
	if !MatchExt("a.CSV", "csv") {
		t.Error("MatchExt 应忽略大小写")
	}
	if MatchExt("a.json", "csv") {
		t.Error("MatchExt 不应跨格式匹配")
	}
}

func TestCSVStripsBOM(t *testing.T) {
	csv := "\xef\xbb\xbfname,age\nAlice,30\n"
	rows, err := csvToRows([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := rows[0]["name"]; !ok {
		t.Fatalf("BOM 未去除导致表头错乱: %v", rows[0])
	}
	if rows[0]["name"] != "Alice" {
		t.Fatalf("got %v", rows[0])
	}
}

func TestCSVSkipsHeaderRow(t *testing.T) {
	csv := "name,age\nAlice,30\n"
	rows, err := csvToRows([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("行数 = %d, want 1（表头不应进数据行）", len(rows))
	}
	if rows[0]["name"] != "Alice" {
		t.Fatalf("got %v", rows[0])
	}
}

func TestJSONSingleObject(t *testing.T) {
	one := `{"k":"v"}`
	rows, err := jsonToRows([]byte(one))
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["k"] != "v" {
		t.Fatalf("单对象应包装为单行: %v", rows)
	}
}

func TestMatchExtCaseInsensitive(t *testing.T) {
	if !MatchExt("report.JSON", "json") {
		t.Fatal("MatchExt 应对扩展名大小写不敏感")
	}
}
