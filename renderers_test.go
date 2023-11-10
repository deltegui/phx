package phx_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/deltegui/phx"
)

type testres struct {
	name       string
	input      interface{}
	err        error
	want       string
	statusCode int
}

type testErr struct {
	Code   int
	Reason string
}

func (t testErr) Error() string {
	return fmt.Sprintf("Code: %d, Reason: %s", t.Code, t.Reason)
}

func TestShouldRenderData(t *testing.T) {
	tt := []testres{
		{
			name:       "Test with data",
			input:      struct{ Name string }{Name: "Manolo"},
			err:        nil,
			want:       "{\"Name\":\"Manolo\"}",
			statusCode: http.StatusOK,
		},
		{
			name:  "Test with data",
			input: struct{ Name string }{Name: "Manolo"},
			err: testErr{
				Code:   0,
				Reason: "First",
			},
			want:       "{\"Code\":0,\"Reason\":\"First\"}",
			statusCode: http.StatusBadRequest,
		},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			recorder := httptest.NewRecorder()

			presenter := phx.NewJsonRenderer(recorder, request)
			presenter(tc.input, tc.err)

			if recorder.Code != tc.statusCode {
				t.Errorf("Want status '%d', got '%d'", tc.statusCode, recorder.Code)
			}

			if strings.TrimSpace(recorder.Body.String()) != tc.want {
				t.Errorf("Want '%s', got '%s'", tc.want, recorder.Body)
			}
		})
	}
}
