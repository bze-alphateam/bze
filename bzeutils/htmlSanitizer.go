package bzeutils

import (
	"github.com/microcosm-cc/bluemonday"

	"sync"
)

var (
	once      sync.Once
	sanitizer *Sanitizer
)

type Sanitizer struct {
	policy *bluemonday.Policy
}

func GetSanitizer() *Sanitizer {
	// Do this once for each unique policy, and use the policy for the life of the program
	// Policy creation/editing is not safe to use in multiple goroutines
	// https://github.com/microcosm-cc/bluemonday#usage
	once.Do(func() {
		sanitizer = &Sanitizer{
			policy: bluemonday.StrictPolicy(),
		}
	})

	return sanitizer
}

func (s Sanitizer) SanitizeHtml(html string) string {
	return s.policy.Sanitize(html)
}
