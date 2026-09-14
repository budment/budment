package tui

import (
	"fmt"
	"strings"

	"github.com/budment/budment/internal/tui/theme"
)

const banner = `
  ╭---╮ ╭-------╮
  ┆   ┆ ┆       ┆
  ┆   ┆ ╰-------╯         ╭-╮                  ╭-╮	                          
  ┆   ┆ ╭------------╮    ┆ ┆                  ┆ ┆                              ╭-╮
  ┆   ┆ ┆            ┆    ┆ ╰----╮╭-╮  ╭-╮╭----╯ ┆╭----------┄╮╭-----╮╭------╮╭-╯ ╰--╮   
  ╰---╯ ╰------------╯    ┆ ╭--╮ ┆┆ ┆  ┆ ┆┆ ╭--╮ ┆┆ ╭--╮ ╭--╮ ┆┆╭----╮┆ ╭--╮ ┆╰-╮ ╭--╯
\----------------------╱  ┆ ╰--╯ ┆┆ ╰--╯ ┆┆ ╰--╯ ┆┆ ┆  ┆ ┆  ┆ ┆┆╰----╯┆ ┆  ┆ ┆  ┆ ╰--╮  
﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿
 \-------------------╱                                                `

func PrintBanner(version string) {
	raw := strings.Trim(banner, "\r\n")
	if raw == "" {
		return
	}
	lines := strings.Split(raw, "\n")
	for i := 0; i < len(lines)-1; i++ {
		fmt.Println(theme.TextCyan(lines[i]))
	}

	lastLine := theme.TextCyan(lines[len(lines)-1])
	versionTag := theme.TextDim("") + theme.TextMagenta(version) + theme.TextDim("")
	authorTag := theme.TextDim("@") + theme.TextMagenta("vunas")
	fmt.Printf("%s   %s %s\n", lastLine, versionTag, authorTag)
}
