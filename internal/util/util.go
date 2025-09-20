package util

import (
	"fmt"
	"log"
	"os"
	// "path/filepath"
	// "runtime"
	"strings"
)

var logger = log.New(os.Stderr, "util: ", log.Lshortfile)

func CloneMap[K comparable, V any](m map[K]V) map[K]V {
	cloned := map[K]V{}

	for k, v := range m {
		cloned[k] = v
	}

	return cloned
}

func CloneMapNested[K1, K2 comparable, V any](m map[K1]map[K2]V) map[K1]map[K2]V {
	cloned := map[K1]map[K2]V{}

	for k, v := range m {
		cloned[k] = CloneMap(v)
	}

	return cloned
}

func CmpMapKeys[K comparable, V any](m1, m2 map[K]V) bool {
	if len(m1) != len(m2) {

		return false
	}

	for k := range m1 {
		if _, ok := m2[k]; !ok {
			return false
		}
	}

	return true
}

func BytesAsHex(b []byte) string {
	var sb strings.Builder

	sb.WriteString("{ ")
	for _, v := range b {
		fmt.Fprintf(&sb, "0x%02x, ", v)
	}
	sb.WriteString("}")

	return sb.String()
}

type HexLogger struct {
	l      *log.Logger
	prefix string
}

func NewHexLogger(l *log.Logger, prefix string) *HexLogger {
	return &HexLogger{l, prefix}
}

func (h *HexLogger) Write(p []byte) (int, error) {
	h.l.Printf("%s: %s", h.prefix, BytesAsHex(p))
	h.l.Printf("%s (string): %q", h.prefix, string(p))
	return len(p), nil
}

func ContainsSubslice[T comparable](haystack []T, needle []T) bool {
	if len(needle) > len(haystack) {
		return false
	}

	start := 0
	end := len(needle)
	for end <= len(haystack) {
		found := true
		for i, ndl := range haystack[start:end] {
			if needle[i] != ndl {
				found = false
				break
			}
		}

		if found {
			return true
		}

		start += 1
		end += 1
	}

	return false
}

func Unwrap[T any](t T, err error) T {
	if err != nil {
		logger.Panic(err)
	}
	return t
}
