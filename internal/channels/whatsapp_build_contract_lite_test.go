//go:build lite

package channels

import (
	"bytes"
	"context"
	"log"
	"strings"
	"testing"

	"github.com/local/picobot/internal/chat"
)

func TestWhatsAppBuildContractLiteUsesStub(t *testing.T) {
	var logs bytes.Buffer
	previousOutput := log.Writer()
	log.SetOutput(&logs)
	t.Cleanup(func() {
		log.SetOutput(previousOutput)
	})

	if err := StartWhatsAppWithOpenMode(context.Background(), chat.NewHub(1), "", nil, false); err != nil {
		t.Fatalf("lite WhatsApp stub returned error = %v, want nil", err)
	}
	if !strings.Contains(logs.String(), "whatsapp: channel not available in 'lite' version") {
		t.Fatalf("lite WhatsApp stub log = %q, want unavailable message", logs.String())
	}

	err := SetupWhatsApp("")
	if err == nil {
		t.Fatal("lite SetupWhatsApp returned nil, want unavailable error")
	}
	if !strings.Contains(err.Error(), "WhatsApp support is not compiled into this binary") {
		t.Fatalf("lite SetupWhatsApp error = %q, want unavailable error", err)
	}
}
