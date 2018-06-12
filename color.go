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
	FColorDefault = 39
	BColorDefault = 49
	isWindows     = runtime.GOOS == "windows"
)

// var isWindows = runtime.GOOS == "windows"

func BlackTextText(str string) string {
	return ColorText(FColorBlack, BColorDefault, str)
}

func RedText(str string) string {
	return ColorText(FColorRed, BColorDefault, str)
}

func GreenText(str string) string {
	return ColorText(FColorGreen, BColorDefault, str)
}

func YellowText(str string) string {
	return ColorText(FColorYellow, BColorDefault, str)
}

func BlueText(str string) string {
	return ColorText(FColorBlue, BColorDefault, str)
}

func MagentaText(str string) string {
	return ColorText(FColorMagenta, BColorDefault, str)
}

func CyanText(str string) string {
	return ColorText(FColorCyan, BColorDefault, str)
}

func WhiteText(str string) string {
	return ColorText(FColorWhite, BColorDefault, str)
}

func ColorText(fcolor, bcolor int, str string) string {
	if isWindows {
		return str
	}
	return fmt.Sprintf("\x1b[%d;%dm%s\x1b[0m", bcolor, fcolor, str)
}
