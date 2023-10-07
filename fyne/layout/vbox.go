package layoutx

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// NewVBoxLayout NewVBoxLayout
func NewVBoxLayout(objects ...fyne.CanvasObject) fyne.CanvasObject {
	return container.New(&VBoxLayout{}, objects...)
}

// VBoxLayout 自定义横向布局
type VBoxLayout struct{}

// MinSize MinSize
func (hb *VBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := theme.Padding()
	for _, child := range objects {
		if !child.Visible() || child.MinSize().Height == 0 {
			continue
		}

		if m := fyne.Max(child.MinSize().Width, child.Size().Width); minSize.Width < m {
			minSize.Width = m
		}
		minSize.Height += fyne.Max(child.MinSize().Height, child.Size().Height)
		if addPadding {
			minSize.Height += padding
		}
		addPadding = true
	}
	return minSize
}

// Layout Layout
func (hb *VBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	total := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child.MinSize().Height == 0 {
			spacers++
			continue
		}
		total += fyne.Max(child.MinSize().Height, child.Size().Height)
	}

	padding := theme.Padding()
	extra := size.Height - total - (padding * float32(len(objects)-spacers-1))
	extraCell := float32(0)
	if spacers > 0 {
		extraCell = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child.MinSize().Height == 0 {
			y += extraCell
			continue
		}
		child.Move(fyne.NewPos(x, y))

		height := fyne.Max(child.MinSize().Height, child.Size().Height)
		y += padding + height
		child.Resize(fyne.NewSize(size.Width, height))
	}
}
