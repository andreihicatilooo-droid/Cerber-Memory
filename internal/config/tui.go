package config

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
)

func RunTUIConfig(useTUI bool) error {
	var (
		geminiKey    string
		masterKey    string
		qdrantHost   string
		qdrantPort   string
	)

	// If TUI is not requested, run in simple CLI mode
	if !useTUI {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("=== Cerber Memory Configuration ===")
		fmt.Println("(Running in simple CLI mode to prevent terminal lag/corruption)")
		fmt.Println("---------------------------------------------------------------")

		fmt.Print("Gemini API Key: ")
		key, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		geminiKey = strings.TrimSpace(key)

		fmt.Print("Master Encryption Key (must be exactly 32 characters for AES-256): ")
		mkey, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		masterKey = strings.TrimSpace(mkey)

		fmt.Print("Qdrant Host: ")
		qhost, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		qdrantHost = strings.TrimSpace(qhost)

		fmt.Print("Qdrant Port: ")
		qport, err := reader.ReadString('\n')
		if err != nil {
			return err
		}
		qdrantPort = strings.TrimSpace(qport)
	} else {
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Gemini API Key").
					Prompt("Key: ").
					Value(&geminiKey).
					Password(true),

				huh.NewInput().
					Title("Master Encryption Key").
					Description("Must be exactly 32 characters for AES-256").
					Prompt("Key: ").
					Value(&masterKey).
					Password(true),
			),
			huh.NewGroup(
				huh.NewInput().
					Title("Qdrant Host").
					Prompt("Host: ").
					Value(&qdrantHost),

				huh.NewInput().
					Title("Qdrant Port").
					Prompt("Port: ").
					Value(&qdrantPort),
			),
		)

		err := form.Run()
		if err != nil {
			return err
		}
	}

	// Merge updates into existing .env to preserve other variables
	err := mergeEnvFile(".env", map[string]string{
		"GEMINI_API_KEY":    geminiKey,
		"CERBER_MASTER_KEY": masterKey,
		"QDRANT_HOST":       qdrantHost,
		"QDRANT_PORT":       qdrantPort,
	})
	if err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	fmt.Println("\nConfiguration saved successfully to .env")
	return nil
}

func mergeEnvFile(path string, updates map[string]string) error {
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = nil
	}

	updated := make(map[string]bool)
	for i, line := range lines {
		if idx := strings.Index(line, "="); idx > 0 {
			k := line[:idx]
			if v, ok := updates[k]; ok && v != "" {
				lines[i] = k + "=" + v
				updated[k] = true
			}
		}
	}

	for k, v := range updates {
		if !updated[k] && v != "" {
			lines = append(lines, k+"="+v)
		}
	}

	return os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
