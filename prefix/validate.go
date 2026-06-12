package prefix

import "fmt"

// ValidateSiblings enforces SPEC §8: no two siblings share a discriminator
// value. Values are compared unpadded ("03" collides with "3"); the meta
// directory is exempt (there can only be one conforming meta dir, and it
// does not extend the prefix).
func ValidateSiblings(discs []Discriminator) error {
	seen := make(map[string]bool, len(discs))
	for _, d := range discs {
		if d.Kind == Meta || d.Value == "" {
			continue
		}
		if seen[d.Value] {
			return fmt.Errorf("%w: %q", ErrDuplicateDiscriminator, d.Value)
		}
		seen[d.Value] = true
	}
	return nil
}
