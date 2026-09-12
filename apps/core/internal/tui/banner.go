package tui

import (
	"fmt"
	"strings"

	"github.com/budment/budment/internal/tui/theme"
)

func PrintBanner() {
	banner := `
	
  ╭---╮ ╭-------╮
  ┆   ┆ ┆       ┆
  ┆   ┆ ╰-------╯            ╭-╮                  ╭-╮	                          
  ┆   ┆ ╭------------╮       ┆ ┆                  ┆ ┆                              ╭-╮
  ┆   ┆ ┆            ┆       ┆ ╰----╮╭-╮  ╭-╮╭----╯ ┆╭----------┄╮╭-----╮╭------╮╭-╯ ╰--╮   
  ┆   ┆ ╰------------╯       ┆ ╭--╮ ┆┆ ┆  ┆ ┆┆ ╭--╮ ┆┆ ╭--╮ ╭--╮ ┆┆╭----╮┆ ╭--╮ ┆╰-╮ ╭--╯
╭-╯   ╰------------------╱   ┆ ╰--╯ ┆┆ ╰--╯ ┆┆ ╰--╯ ┆┆ ┆  ┆ ┆  ┆ ┆┆╰----╯┆ ┆  ┆ ┆  ┆ ╰--╮  
﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿﹀︿
╰----------------------╱                                                           `

	lines := strings.Split(strings.TrimPrefix(banner, "\n"), "\n")

	for i := 0; i < len(lines)-1; i++ {
		fmt.Println(theme.TextCyan(lines[i]))
	}

	lastLine := theme.TextCyan(lines[len(lines)-1])
	authorTag := theme.TextDim("@") + theme.TextMagenta("vunas")
	fmt.Printf("%s%s\n", lastLine, authorTag)
	fmt.Printf("\n")
}
