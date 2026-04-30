//go:build !lite

package channels

import (
	"context"
	"strings"
	"testing"

	"github.com/local/picobot/internal/chat"
)

func TestWhatsAppBuildContractFullUsesRealImplementation(t *testing.T) {
	err := StartWhatsAppWithOpenMode(context.Background(), chat.NewHub(1), "", nil, false)
	if err == nil {
		t.Fatal("full build used lite WhatsApp stub, want real implementation")
	}
	if !strings.Contains(err.Error(), "whatsapp database path not provided") {
		t.Fatalf("full build WhatsApp error = %q, want db path validation", err)
	}

	err = SetupWhatsApp("")
	if err == nil {
		t.Fatal("full build SetupWhatsApp returned nil for empty db path")
	}
	if !strings.Contains(err.Error(), "whatsapp database path not provided") {
		t.Fatalf("full build SetupWhatsApp error = %q, want db path validation", err)
	}
}
