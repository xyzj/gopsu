// Package widgetx 自定义组件
package widgetx

import (
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/widget"
)

// ComboxEntry ComSelect
func ComboxEntry(text string, option ...string) *widget.SelectEntry {
	w := widget.NewSelectEntry(option)
	w.SetText(text)
	return w
}

// ComboxSelect ComModule
func ComboxSelect(idx int, option ...string) *widget.Select {
	w := widget.NewSelect(option, func(s string) {})
	w.SetSelectedIndex(idx)
	return w
}

// NumberEntryPI NumberEntry
func NumberEntryPI(text int) *widget.Entry {
	w := widget.NewEntry()
	w.SetText(strconv.Itoa(text))
	w.Validator = validation.NewRegexp(`^\+?[1-9][0-9]*$`, "must be a positive integer")
	return w
}

// StrEntry StrEntry
func StrEntry(text string) *widget.Entry {
	w := widget.NewEntry()
	w.SetText(text)
	return w
}

// RightAlignLabel StrLabel
func RightAlignLabel(text string) *widget.Label {
	w := widget.NewLabelWithStyle(text, fyne.TextAlignTrailing, fyne.TextStyle{})
	return w
}
