package queue

import (
	"testing"

	"github.com/renankeny/conversion-tracker/backend/internal/domain"
)

func TestIsPermanentResult(t *testing.T) {
	cases := []struct {
		name string
		res  dispatchResult
		want bool
	}{
		{
			name: "400 validation error is permanent",
			res:  dispatchResult{StatusCode: 400, Body: `{"error":{"code":100,"message":"Invalid parameter"}}`},
			want: true,
		},
		{
			name: "400 meta rate limit code 17 is retryable",
			res:  dispatchResult{StatusCode: 400, Body: `{"error":{"code":17,"message":"User request limit reached"}}`},
			want: false,
		},
		{
			name: "403 meta rate limit code 613 is retryable",
			res:  dispatchResult{StatusCode: 403, Body: `{"error":{"code":613,"message":"Calls to this api have exceeded the rate limit"}}`},
			want: false,
		},
		{
			name: "400 meta transient error is retryable",
			res:  dispatchResult{StatusCode: 400, Body: `{"error":{"code":2,"is_transient":true}}`},
			want: false,
		},
		{
			name: "401 unauthorized is permanent",
			res:  dispatchResult{StatusCode: 401, Body: `{"error":{"code":190,"message":"Invalid OAuth access token"}}`},
			want: true,
		},
		{
			name: "500 internal error is retryable",
			res:  dispatchResult{StatusCode: 500, Body: `internal error`},
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isPermanentResult(tc.res); got != tc.want {
				t.Fatalf("isPermanentResult(%+v) = %v, want %v", tc.res, got, tc.want)
			}
		})
	}
}

func TestRecoverDispatchPanic(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer recoverDispatchPanic("meta", domain.SendEventInput{PixelName: "px", EventName: "Purchase", EventID: "ev1"})
		panic("boom")
	}()
	<-done
}

