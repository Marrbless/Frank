package modelsetup

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type LlamaCPPRegistration struct {
	ServerPath  string
	ModelPath   string
	BindAddress string
	Port        int
	Command     string
}

func BuildLlamaCPPRegistration(serverPath, modelPath, bindAddress string, port int) (LlamaCPPRegistration, error) {
	serverPath = strings.TrimSpace(serverPath)
	modelPath = strings.TrimSpace(modelPath)
	bindAddress = strings.TrimSpace(bindAddress)
	if serverPath == "" {
		return LlamaCPPRegistration{}, fmt.Errorf("llama.cpp server path is required")
	}
	if modelPath == "" {
		return LlamaCPPRegistration{}, fmt.Errorf("GGUF model path is required")
	}
	if bindAddress == "" {
		bindAddress = "127.0.0.1"
	}
	if bindAddress != "127.0.0.1" {
		return LlamaCPPRegistration{}, fmt.Errorf("llama.cpp register-existing defaults must bind to 127.0.0.1, got %q", bindAddress)
	}
	if port <= 0 {
		return LlamaCPPRegistration{}, fmt.Errorf("llama.cpp port is required")
	}
	return LlamaCPPRegistration{
		ServerPath:  serverPath,
		ModelPath:   modelPath,
		BindAddress: bindAddress,
		Port:        port,
		Command:     shellJoin([]string{serverPath, "-m", modelPath, "--host", bindAddress, "--port", strconv.Itoa(port)}),
	}, nil
}

func ValidateLlamaCPPRegistration(reg LlamaCPPRegistration) error {
	if reg.ServerPath == "" || reg.ModelPath == "" {
		return fmt.Errorf("llama.cpp registration paths are required")
	}
	if reg.BindAddress != "127.0.0.1" {
		return fmt.Errorf("llama.cpp registration bind address %q is not localhost", reg.BindAddress)
	}
	if reg.Port <= 0 {
		return fmt.Errorf("llama.cpp registration port is required")
	}
	if err := requireExistingFile(reg.ServerPath, "llama.cpp server"); err != nil {
		return err
	}
	if err := requireExistingFile(reg.ModelPath, "GGUF model"); err != nil {
		return err
	}
	return nil
}

func requireExistingFile(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("%s path %q is not available: %w", label, path, err)
	}
	if info.IsDir() {
		return fmt.Errorf("%s path %q is a directory", label, path)
	}
	return nil
}
