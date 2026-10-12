package contract

import (
	"fmt"
	"strings"
)

// ValidateToolIdentity validates an authoring ProductID.Name identity part.
func ValidateToolIdentity(productID, name string) error {
	if err := ValidateToolIdentityPart("ProductID", productID); err != nil {
		return err
	}
	return ValidateToolIdentityPart("Name", name)
}

func ValidateToolIdentityPart(label, value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return fmt.Errorf("%s is empty", label)
	}
	for index, r := range value {
		if index == 0 {
			if !isASCIIIdentityLetter(r) {
				return fmt.Errorf("%s %q must start with an ASCII letter", label, value)
			}
			continue
		}
		if !isASCIIIdentityLetter(r) && !isASCIIIdentityDigit(r) && r != '_' && r != '-' {
			return fmt.Errorf("%s %q contains invalid character %q", label, value, r)
		}
	}
	return nil
}

func isASCIIIdentityLetter(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

func isASCIIIdentityDigit(r rune) bool {
	return r >= '0' && r <= '9'
}
