// Helper package containing miscellaneous functions
package helper

import (
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"time"
)

// TimeWithoutNanoseconds is a wrapper for time.time, intended to represent a time value without nanoseconds.
type TimeWithoutNanoseconds struct {
	time.Time
}

// MarshalJSON override the default JSON Marshalling for TimeWithoutNanoseconds.
// Returns a JSON byte array encoding time formatted without nanoseconds
func (t TimeWithoutNanoseconds) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, t.Format("2006-01-02T15:04:05-07:00"))), nil
}

// SHAFromString computes a hash representation from an URL string
// Return a base64 encoded string representation
func SHAFromString(urlstring string) string {
	bv := []byte(urlstring)
	hasher := sha1.New()
	hasher.Write(bv)
	sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))
	return sha
}
