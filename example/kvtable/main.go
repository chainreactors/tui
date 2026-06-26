package main

import (
	"time"

	"github.com/chainreactors/tui"
)

type User struct {
	Name string
	Age  int
}

func main() {
	// 创建一个包含各种数据类型的示例数据
	slice := []string{"apple", "banana", "orange"}
	nestedMap := map[string]int{"a": 1, "b": 2}
	user := User{Name: "Alice", Age: 25}
	var ptr *string
	str := "hello"
	ptr = &str

	data := map[string]interface{}{
		// 基本类型
		"字符串": "这是一个字符串", // 白色
		"数字":  42,        // 青色
		"浮点数": 3.14159,   // 青色
		"布尔值": true,      // 绿色

		// 复合类型 (都使用黄色)
		"切片":  slice,
		"映射":  nestedMap,
		"结构体": user,
		"指针":  ptr,

		// 特殊值
		"空指针":  (*string)(nil),
		"空切片":  []int{},
		"当前时间": time.Now(),
		"空字符串": "",
	}

	// 直接渲染，无需等待用户输入
	tui.RenderKV(data)
}
