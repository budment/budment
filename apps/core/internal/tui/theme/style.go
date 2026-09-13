package theme

import "strings"

func Apply(text string, codes ...string) string {
	if !ColorEnabled {
		return text
	}
	return strings.Join(codes, "") + text + Reset
}

func TextRed(t string) string     { return Apply(t, Red) }
func TextGreen(t string) string   { return Apply(t, Green) }
func TextYellow(t string) string  { return Apply(t, Yellow) }
func TextBlue(t string) string    { return Apply(t, Blue) }
func TextCyan(t string) string    { return Apply(t, Cyan) }
func TextMagenta(t string) string { return Apply(t, Magenta) }
func TextGray(t string) string    { return Apply(t, Gray) }
func TextBold(t string) string    { return Apply(t, Bold) }
func TextDim(t string) string     { return Apply(t, Dim) }
func TextItalic(t string) string  { return Apply(t, Italic) }

func Success(t string) string { return Apply(t, Green, Bold) }
func Warning(t string) string { return Apply(t, Yellow, Bold) }
func Error(t string) string   { return Apply(t, Red, Bold) }
func Info(t string) string    { return Apply(t, Cyan) }

func Brand(t string) string    { return Apply(t, Cyan, Bold) }
func Subtitle(t string) string { return Apply(t, Gray, Italic) }
