// Package themex 支持中文的主题
package themex

import (
	// 静态
	_ "embed"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

//go:embed favicon.png
var favicon []byte

//go:embed DroidSansFallbackFull.ttf
var cnfonts []byte

// Favicon 图标
func Favicon() *fyne.StaticResource {
	return fyne.NewStaticResource("favicon.png", favicon)
}

// ZhHans 中文主题
type ZhHans struct{}

var _ fyne.Theme = (*ZhHans)(nil)

// Color Color
func (m ZhHans) Color(n fyne.ThemeColorName, v fyne.ThemeVariant) color.Color {
	if n == "disabled" {
		return color.RGBA{R: 77, G: 77, B: 77, A: 200}
	}
	return theme.DefaultTheme().Color(n, v)
}

// Icon Icon
func (m ZhHans) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

// Font Font
func (m ZhHans) Font(style fyne.TextStyle) fyne.Resource {
	//return theme.DefaultTheme().Font(style)
	return &fyne.StaticResource{
		StaticName:    "DroidSansFallbackFull.ttf",
		StaticContent: cnfonts,
	}
}

// Size Size
func (m ZhHans) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}
