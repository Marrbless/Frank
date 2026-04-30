package channels

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

var logSecretPatterns = []struct {
	pattern     *regexp.Regexp
	replacement string
}{
	{regexp.MustCompile(`(?i)Bearer\s+[A-Za-z0-9._~+/=-]+`), `Bearer <redacted-token>`},
	{regexp.MustCompile(`xox[baprs]-[A-Za-z0-9-]+`), `<redacted-token>`},
	{regexp.MustCompile(`xapp-[A-Za-z0-9-]+`), `<redacted-token>`},
	{regexp.MustCompile(`sk-[A-Za-z0-9_-]+`), `<redacted-token>`},
	{regexp.MustCompile(`bot[^/\s"]+`), `bot<redacted-token>`},
}

func summarizeInboundContent(content string, attachmentCount int) string {
	return fmt.Sprintf("chars=%d attachments=%d", utf8.RuneCountInString(content), attachmentCount)
}

func redactLogID(id string) string {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return "redacted:empty"
	}
	sum := sha256.Sum256([]byte(trimmed))
	return "redacted:" + hex.EncodeToString(sum[:4])
}

func redactLogText(text string) string {
	redacted := text
	for _, secretPattern := range logSecretPatterns {
		redacted = secretPattern.pattern.ReplaceAllString(redacted, secretPattern.replacement)
	}
	return redacted
}

func redactLogError(err error) string {
	if err == nil {
		return ""
	}
	return redactLogText(err.Error())
}
