package backend

import (
	"fmt"
	"log"
	"os"
	"strings"
)

func gatheringDir(gatheringID uint) string {
	return fmt.Sprintf("/tmp/must-gather-result-%d", gatheringID)
}

// TODO: Consider make or use something nicer
func sanitizeArg(str string) string {
	output := ""
	allowedChars := "QWERTZUIOPASDFGHJKLYXCVBNMqwertzuiopasdfghjklyxcvbnm1234567890_-~/:.=@ "
	for _, rune := range str {
		if strings.ContainsRune(allowedChars, rune) {
			output += string(rune)
		}
	}
	if len(output) < len(str) {
		log.Printf("Warning: sanitizeArg function modified its input to: %s", output)
	}
	return output
}

func ConfigEnvOrDefault(name, defaultValue string) string {
	value, present := os.LookupEnv(name)
	if present {
		log.Printf("Config option %s set from environment variable to: %s", name, value)
		return value
	} else {
		log.Printf("Environment variable %s is undefined, using default %s", name, defaultValue)
		return defaultValue
	}
}
