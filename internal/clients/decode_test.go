package clients

import (
	"errors"
	"testing"
)

func TestDecodeID(t *testing.T) {
	cases := []struct {
		body string
		want int
		ok   bool
	}{
		{`3`, 3, true},
		{`"3"`, 3, true},
		{`{"value":3}`, 3, true},
		{`{"value":"3"}`, 3, true},
		{`  7 `, 7, true},
		{`"nope"`, 0, false},
		{`{}`, 0, false},
	}
	for _, tc := range cases {
		got, err := DecodeID([]byte(tc.body))
		if tc.ok && (err != nil || got != tc.want) {
			t.Fatalf("DecodeID(%s)=%d,%v want %d", tc.body, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Fatalf("DecodeID(%s) expected error", tc.body)
		}
	}
}

func TestDecodeString(t *testing.T) {
	cases := []struct {
		body string
		want string
		ok   bool
	}{
		{`"abc123"`, "abc123", true},
		{`{"value":"tok"}`, "tok", true},
		{`[1,2]`, "", false},
	}
	for _, tc := range cases {
		got, err := DecodeString([]byte(tc.body))
		if tc.ok && (err != nil || got != tc.want) {
			t.Fatalf("DecodeString(%s)=%q,%v want %q", tc.body, got, err, tc.want)
		}
		if !tc.ok && err == nil {
			t.Fatalf("DecodeString(%s) expected error", tc.body)
		}
	}
}

func TestIsNotFound(t *testing.T) {
	if !IsNotFound(errors.New("matomo API error: The requested website id = '5' couldn't be found")) {
		t.Fatal("expected not-found match")
	}
	if IsNotFound(errors.New("matomo API error: You can't access this resource")) {
		t.Fatal("unexpected not-found match")
	}
	if IsNotFound(nil) {
		t.Fatal("nil must not be not-found")
	}
}
