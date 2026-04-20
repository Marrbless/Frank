package channels

import (
	"fmt"
	"unicode/utf8"
)

func summarizeInboundContent(content string, attachmentCount int) string {
	return fmt.Sprintf("chars=%d attachments=%d", utf8.RuneCountInString(content), attachmentCount)
}
