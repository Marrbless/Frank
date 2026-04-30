package channels

import (
	"fmt"
	"strings"
)

func requireAllowlistOrExplicitOpen(surface string, allowed []string, openMode bool) error {
	if len(allowed) > 0 || openMode {
		return nil
	}
	return fmt.Errorf("%s allowlist is empty; set an allowlist or enable explicit open mode", strings.TrimSpace(surface))
}

func buildAllowedSet(allowed []string) map[string]struct{} {
	set := make(map[string]struct{}, len(allowed))
	for _, id := range allowed {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		set[id] = struct{}{}
	}
	return set
}

func allowedBySingleAllowlist(allowed map[string]struct{}, openMode bool, id string) bool {
	if len(allowed) == 0 {
		return openMode
	}
	_, ok := allowed[id]
	return ok
}
