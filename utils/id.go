package utils

import (
	"crypto/rand"
	"strings"
)

const idChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// IDLength is the number of characters in a generated task ID.
const IDLength = 5

// GenerateID creates a random lowercase alphanumeric string of the given length.
func GenerateID(length int) string {
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	for i := range b {
		b[i] = idChars[int(b[i])%len(idChars)]
	}
	return string(b)
}

// FormatTaskID returns "project-id".
func FormatTaskID(project, id string) string {
	return project + "-" + id
}

// ParseTaskID splits input into (project, id).
// Accepts both "project-k7x2" and bare "k7x2".
func ParseTaskID(input string) (project, id string) {
	idx := strings.LastIndex(input, "-")
	if idx == -1 {
		return "", input
	}
	return input[:idx], input[idx+1:]
}
