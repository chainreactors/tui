package tui

import (
	"fmt"
	"reflect"

	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

var (
	// kvTableStyle 定义 key-value 表格的基本样式
	kvTableStyle = lipgloss.NewStyle().
			Align(lipgloss.Left)

	// keyStyle 定义键的样式
	keyStyle = lipgloss.NewStyle().
			Foreground(Blue).
			Bold(true).
			PaddingRight(2)
)

// getValueStyle 根据值的类型返回对应的样式
func getValueStyle(v interface{}) lipgloss.Style {
	switch reflect.TypeOf(v).Kind() {
	case reflect.String:
		return WhiteFg // 字符串使用白色
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return CyanFg // 数字类型使用青色
	case reflect.Bool:
		return GreenFg // 布尔值使用绿色
	case reflect.Slice, reflect.Array, reflect.Map, reflect.Struct, reflect.Ptr:
		return YellowFg // 复合类型统一使用黄色
	default:
		return WhiteFg // 其他类型使用白色
	}
}

// formatValue 格式化值并添加类型信息（如果需要）
func formatValue(v interface{}) string {
	if v == nil {
		return "nil"
	}

	t := reflect.TypeOf(v)
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		val := reflect.ValueOf(v)
		if val.Len() == 0 {
			return "[]"
		}
		return fmt.Sprintf("%v", v)
	case reflect.Map:
		val := reflect.ValueOf(v)
		if val.Len() == 0 {
			return "{}"
		}
		return fmt.Sprintf("%v", v)
	case reflect.Struct:
		return fmt.Sprintf("%+v", v)
	case reflect.Ptr:
		if reflect.ValueOf(v).IsNil() {
			return "nil"
		}
		return fmt.Sprintf("%v", reflect.ValueOf(v).Elem())
	default:
		return fmt.Sprintf("%v", v)
	}
}

// NewKVTable 创建一个新的 key-value 表格
func NewKVTable(data map[string]interface{}) *TableModel {
	// 定义列
	columns := []table.Column{
		table.NewColumn("key", "Key", 20),
		table.NewColumn("value", "Value", 40),
	}

	// 创建表格模型
	t := NewTable(columns, true)
	t.table = t.table.
		WithBaseStyle(kvTableStyle).
		WithHeaderVisibility(false)

	// 转换数据为表格行
	var rows []table.Row
	for k, v := range data {
		valueStyle := getValueStyle(v)
		row := table.NewRow(table.RowData{
			"key":   keyStyle.Render(k),
			"value": valueStyle.Render(formatValue(v)),
		})
		rows = append(rows, row)
	}

	// 设置行数据
	t.SetRows(rows)
	return t
}

func NewOrderedKVTable(data map[string]interface{}, orderedKeys []string) *TableModel {
	// 定义列
	columns := []table.Column{
		table.NewColumn("key", "Key", 20),
		table.NewColumn("value", "Value", 40),
	}

	// 创建表格模型
	t := NewTable(columns, true)
	t.table = t.table.
		WithBaseStyle(kvTableStyle).
		WithHeaderVisibility(false)

	// 转换数据为表格行，按指定的键顺序
	var rows []table.Row
	for _, k := range orderedKeys {
		if v, exists := data[k]; exists {
			valueStyle := getValueStyle(v)
			row := table.NewRow(table.RowData{
				"key":   keyStyle.Render(k),
				"value": valueStyle.Render(formatValue(v)),
			})
			rows = append(rows, row)
		}
	}

	// 设置行数据
	t.SetRows(rows)
	return t
}

// RenderKV 直接渲染 key-value 数据，渲染完立即返回
func RenderKV(data map[string]interface{}, orderedKeys []string) {
	table := NewOrderedKVTable(data, orderedKeys)
	fmt.Println(table.View())
}
