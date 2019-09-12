package gopsu

import (
	"fmt"
	"runtime"
)

const (
	// FColorBlack 字体颜色黑色
	FColorBlack = iota + 30
	// FColorRed 字体颜色红色
	FColorRed
	// FColorGreen 字体颜色绿色
	FColorGreen
	// FColorYellow 字体颜色黄色
	FColorYellow
	// FColorBlue 字体颜色蓝色
	FColorBlue
	// FColorMagenta 字体颜色品红
	FColorMagenta
	// FColorCyan 字体颜色青色
	FColorCyan
	// FColorWhite 字体颜色白色
	FColorWhite
)

const (
	// BColorBlack 背景色
	BColorBlack = iota + 40
	// BColorRed 背景色
	BColorRed
	// BColorGreen 背景色
	BColorGreen
	// BColorYellow 背景色
	BColorYellow
	// BColorBlue 背景色
	BColorBlue
	// BColorMagenta 背景色
	BColorMagenta
	// BColorCyan 背景色
	BColorCyan
	// BColorWhite 背景色
	BColorWhite
)

const (
	// TextNormal 终端默认设置
	TextNormal = 0
	// TextHighlight 1  高亮显示
	TextHighlight = 1
	// TextUnderline 4  使用下划线
	TextUnderline = 4
	// TextFlicker 5  闪烁
	TextFlicker = 5
	// TextAntiWhite 7  反白显示
	TextAntiWhite = 7
	// TextUnvisiable 8  不可见
	TextUnvisiable = 8
)
const (
	// FColorDefault 默认字体色
	FColorDefault = 39
	// BColorDefault 默认背景色
	BColorDefault = 49
	isWindows     = runtime.GOOS == "windows"
)

// var isWindows = runtime.GOOS == "windows"

// BlackText 黑字
func BlackText(str string) string {
	return ColorText(FColorBlack, BColorDefault, TextNormal, str)
}

// RedText 红字
func RedText(str string) string {
	return ColorText(FColorRed, BColorDefault, TextNormal, str)
}

// GreenText 绿字
func GreenText(str string) string {
	return ColorText(FColorGreen, BColorDefault, TextNormal, str)
}

// YellowText 黄字
func YellowText(str string) string {
	return ColorText(FColorYellow, BColorDefault, TextNormal, str)
}

// BlueText 蓝字
func BlueText(str string) string {
	return ColorText(FColorBlue, BColorDefault, TextNormal, str)
}

// MagentaText 品红字
func MagentaText(str string) string {
	return ColorText(FColorMagenta, BColorDefault, TextNormal, str)
}

// CyanText 青色字
func CyanText(str string) string {
	return ColorText(FColorCyan, BColorDefault, TextNormal, str)
}

// WhiteText 白色文字
func WhiteText(str string) string {
	return ColorText(FColorWhite, BColorDefault, TextNormal, str)
}

// ColorText 自定义色彩文字
func ColorText(fcolor, bcolor, textstyle int, str string) string {
	if isWindows {
		return str
	}
	return fmt.Sprintf("\x1b[%d;%d;%dm%s\x1b[0m", textstyle, bcolor, fcolor, str)
}
