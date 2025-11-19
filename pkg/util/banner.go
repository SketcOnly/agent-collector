package util

import (
	"fmt"
	"github.com/common-nighthawk/go-figure"
)

// 定义颜色常量
const (
	ColorReset  = "\x1b[0m"
	ColorRed    = "\x1b[1;31m"
	ColorGreen  = "\x1b[1;32m"
	ColorYellow = "\x1b[1;33m"
	ColorBlue   = "\x1b[1;34m"
	ColorCyan   = "\x1b[1;36m"
)

// 字符串转 ANSI 颜色码
func colorCode(name string) string {
	switch name {
	case "ColorRed":
		return ColorRed
	case "ColorGreen":
		return ColorGreen
	case "ColorYellow":
		return ColorYellow
	case "ColorBlue":
		return ColorBlue
	case "ColorCyan":
		return ColorCyan
	default:
		return ColorReset
	}
}

// PrintBanner 打印整体统一颜色的 ASCII banner
func PrintBanner(text string, color string) {
	fig := figure.NewFigure(text, "", true)
	lines := fig.Slicify() // 获取每行 ASCII 字符

	ansiColor := colorCode(color)
	for _, line := range lines {
		fmt.Println(ansiColor + line + ColorReset)
	}
}
