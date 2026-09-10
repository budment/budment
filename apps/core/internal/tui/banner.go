package tui

import (
	"fmt"
	"strings"

	"github.com/budment/budment/internal/tui/theme"
)

func PrintBanner() {
	banner := `
 ____  _        _    ____ _____ _____ ____  
| __ )| |      / \  / ___|_   _| ____|  _ \ 
|  _ \| |     / _ \ \___ \ | | |  _| | |_) |
| |_) | |___ / ___ \ ___) || | | |___|  _ < 
|____/|_____/_/   \_\____/ |_| |_____|_| \_\`

	lines := strings.Split(strings.TrimPrefix(banner, "\n"), "\n")

	for i := 0; i < len(lines)-1; i++ {
		fmt.Println(theme.TextCyan(lines[i]))
	}

	lastLine := theme.TextCyan(lines[len(lines)-1])
	authorTag := theme.TextDim("  by ") + theme.TextMagenta("vunas")
	fmt.Printf("%s%s\n", lastLine, authorTag)
}
