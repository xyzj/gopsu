package mxgo

import (
	"fmt"
	"runtime"
)

const (
	FColorBlack = iota + 30
	FColorRed
	FColorGreen
	FColorYellow
	FColorBlue
	FColorMagenta
	FColorCyan
	FColorWhite
)

const (
	BColorBlack = iota + 40
	BColorRed
	BColorGreen
	BColorYellow
	BColorBlue
	BColorMagenta
	BColorCyan
	BColorWhite
)

const (
	TextNormal     = 0 // 终端默认设置
	TextHighlight  = 1 //  1  高亮显示
	TextUnderline  = 4 //  4  使用下划线
	TextFlicker    = 5 //  5  闪烁
	TextAntiWhite  = 7 //  7  反白显示
	TextUnvisiable = 8 //  8  不可见
)
const (
	FColorDefault = 39
	BColorDefault = 49
	isWindows     = runtime.GOOS == "windows"
)

// var isWindows = runtime.GOOS == "windows"

func BlackTextText(str string) string {
	return ColorText(FColorBlack, BColorDefault, TextNormal, str)
}

func RedText(str string) string {
	return ColorText(FColorRed, BColorDefault, TextNormal, str)
}

func GreenText(str string) string {
	return ColorText(FColorGreen, BColorDefault, TextNormal, str)
}

func YellowText(str string) string {
	return ColorText(FColorYellow, BColorDefault, TextNormal, str)
}

func BlueText(str string) string {
	return ColorText(FColorBlue, BColorDefault, TextNormal, str)
}

func MagentaText(str string) string {
	return ColorText(FColorMagenta, BColorDefault, TextNormal, str)
}

func CyanText(str string) string {
	return ColorText(FColorCyan, BColorDefault, TextNormal, str)
}

func WhiteText(str string) string {
	return ColorText(FColorWhite, BColorDefault, TextNormal, str)
}

func ColorText(fcolor, bcolor, textstyle int, str string) string {
	if isWindows {
		return str
	}
	return fmt.Sprintf("\x1b[%d;%d;%dm%s\x1b[0m", textstyle, bcolor, fcolor, str)
}
