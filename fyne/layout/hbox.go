// Package layoutx custom layout
package layoutx

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// NewHBoxLayout NewHBoxLayout
func NewHBoxLayout(objects ...fyne.CanvasObject) fyne.CanvasObject {
	return container.New(&HBoxLayout{}, objects...)
}

// HBoxLayout 自定义横向布局
type HBoxLayout struct{}

// MinSize MinSize
func (hb *HBoxLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	minSize := fyne.NewSize(0, 0)
	addPadding := false
	padding := theme.Padding()
	for _, child := range objects {
		if !child.Visible() || child.MinSize().Width == 0 {
			continue
		}

		childMin := child.MinSize()
		if m := fyne.Max(child.MinSize().Height, child.Size().Height); minSize.Height < m {
			minSize.Height = m
		}
		minSize.Width += fyne.Max(childMin.Width, child.Size().Width)

		// minSize.Height = fyne.Max(childMin.Height, minSize.Height)
		// minSize.Width += childMin.Width
		if addPadding {
			minSize.Width += padding
		}
		addPadding = true
	}
	return minSize
}

// Layout Layout
func (hb *HBoxLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	spacers := 0
	total := float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child.MinSize().Width == 0 {
			spacers++
			continue
		}
		total += fyne.Max(child.MinSize().Width, child.Size().Width)
	}

	padding := theme.Padding()
	extra := size.Width - total - (padding * float32(len(objects)-spacers-1))
	extraCell := float32(0)
	if spacers > 0 {
		extraCell = extra / float32(spacers)
	}

	x, y := float32(0), float32(0)
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		if child.MinSize().Width == 0 {
			x += extraCell
			continue
		}
		child.Move(fyne.NewPos(x, y))

		width := fyne.Max(child.MinSize().Width, child.Size().Width)
		x += padding + width
		child.Resize(fyne.NewSize(width, size.Height))
	}
}
