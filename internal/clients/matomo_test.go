package clients

import "testing"

func TestParseResponse(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{"error envelope", `{"result":"error","message":"You can't access this resource"}`, true},
		{"success envelope", `{"result":"success","message":"ok"}`, false},
		{"object result", `{"idsite":"3"}`, false},
		{"array result", `[{"idsite":"1"},{"idsite":"2"}]`, false},
		{"bare number", `7`, false},
		{"leading whitespace error", "  \n{\"result\":\"error\",\"message\":\"nope\"}", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseResponse([]byte(tc.body))
			if tc.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
