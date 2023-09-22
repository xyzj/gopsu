// Package widgetx 自定义组件
package widgetx

import (
	"strconv"
	"strings"

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

// NumberEntry NumberEntry
func NumberEntry(text int, placeholder ...string) *widget.Entry {
	w := widget.NewEntry()
	w.SetText(strconv.Itoa(text))
	w.SetPlaceHolder(strings.Join(placeholder, "\n"))
	w.Validator = validation.NewRegexp(`^[0-9]*$`, "must be an unsigned integer")
	return w
}

// NumberEntryPI NumberEntry
func NumberEntryPI(text int, placeholder ...string) *widget.Entry {
	w := widget.NewEntry()
	w.SetText(strconv.Itoa(text))
	w.SetPlaceHolder(strings.Join(placeholder, "\n"))
	w.Validator = validation.NewRegexp(`^\+?[1-9][0-9]*$`, "must be a positive integer")
	return w
}

// StrEntry StrEntry
func StrEntry(text string, placeholder ...string) *widget.Entry {
	w := widget.NewEntry()
	w.SetText(text)
	w.SetPlaceHolder(strings.Join(placeholder, "\n"))
	return w
}

// RightAlignLabel StrLabel
func RightAlignLabel(text string) *widget.Label {
	w := widget.NewLabelWithStyle(text, fyne.TextAlignTrailing, fyne.TextStyle{})
	return w
}

// MultiLineEntry MultiLineEntry
func MultiLineEntry(visibleRows, maxRows int, wrap fyne.TextWrap) *widget.Entry {
	w := widget.NewMultiLineEntry()
	w.SetMinRowsVisible(visibleRows)
	w.Wrapping = wrap
	w.OnChanged = func(s string) {
		if maxRows == 0 {
			return
		}
		ss := strings.Split(s, "\n")
		l := len(ss)
		if l > maxRows {
			w.SetText(strings.Join(ss[l-maxRows:], "\n"))
		}
		w.CursorRow = l + 1
	}
	return w
}

// Checkbox Checkbox
func Checkbox(text string, checked bool) *widget.Check {
	w := widget.NewCheck(text, func(b bool) {})
	w.SetChecked(checked)
	return w
}
