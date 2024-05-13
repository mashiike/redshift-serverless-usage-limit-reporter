package reporter

import (
	"encoding/json"
	"os"
	"testing"
)

func TestMessageIsRedshiftServerlessLimitUsage(t *testing.T) {
	cases := []struct {
		file   string
		expect bool
	}{
		{
			file:   "testdata/redshift_serverless_limit_usage_payload.json",
			expect: true,
		},
		{
			file:   "testdata/example_paylaod.json",
			expect: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.file, func(t *testing.T) {
			fp, err := os.Open(tc.file)
			if err != nil {
				t.Fatal(err)
			}
			defer fp.Close()
			dec := json.NewDecoder(fp)
			var message CloudWatchAlermMessage
			if err := dec.Decode(&message); err != nil {
				t.Fatal(err)
			}
			if got, want := message.IsRedshiftServerlessLimitUsage(), tc.expect; got != want {
				t.Errorf("unexpected result; got = %v, want = %v", got, want)
			}
		})
	}
}
