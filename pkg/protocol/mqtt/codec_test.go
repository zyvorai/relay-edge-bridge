package mqtt

import (
	"bufio"
	"bytes"
	"testing"
)

func TestRemainingLength(t *testing.T) {
	for _, n := range []int{0, 1, 127, 128, 16384, 2097151} {
		b := encodeRemaining(n)
		got, err := readRemaining(bufio.NewReader(bytes.NewReader(b)))
		if err != nil || got != n {
			t.Fatalf("n=%d got=%d err=%v", n, got, err)
		}
	}
}
func TestTopicMatch(t *testing.T) {
	cases := []struct {
		f, t string
		ok   bool
	}{{"a/+/c", "a/b/c", true}, {"a/#", "a/b/c", true}, {"a/b", "a/b/c", false}, {"a/+/c", "a/b/d", false}}
	for _, c := range cases {
		if got := topicMatch(c.f, c.t); got != c.ok {
			t.Fatalf("%s %s: %v", c.f, c.t, got)
		}
	}
}
