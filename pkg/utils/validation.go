package utils

import (
	"strings"
	"unicode"
)

// ValidateUsername checks if a username is valid
func ValidateUsername(username string) bool {
	if len(username) < 3 || len(username) > 20 {
		return false
	}
	
	for _, char := range username {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' && char != '-' {
			return false
		}
	}
	
	return true
}

// ValidatePassword checks if a password meets minimum requirements
func ValidatePassword(password string) bool {
	if len(password) < 6 {
		return false
	}
	return true
}

// SanitizeString removes whitespace and converts to lowercase
func SanitizeString(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}