package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestIdentify(t *testing.T) {
	type request struct {
		headers map[string]string
	}
	type response struct {
		code int
		body string
	}

	tests := []struct {
		description string
		input       request
		want        response
	}{
		{
			description: "missing header",
			input: request{
				headers: map[string]string{},
			},
			want: response{
				code: http.StatusBadRequest,
				body: `{"errors":[{"status":"Bad Request","title":"missing X-Rh-Identity header"}]}`,
			},
		},
		{
			description: "empty header",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": "",
				},
			},
			want: response{
				code: http.StatusBadRequest,
				body: `{"errors":[{"status":"Bad Request","title":"missing X-Rh-Identity header"}]}`,
			},
		},
		{
			description: "bad base64",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": "0xdeadbeef",
				},
			},
			want: response{
				code: http.StatusBadRequest,
				body: `{"errors":[{"status":"Bad Request","title":"illegal base64 data at input byte 8"}]}`,
			},
		},
		{
			description: "bad json",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{`)),
				},
			},
			want: response{
				code: http.StatusBadRequest,
				body: `{"errors":[{"status":"Bad Request","title":"unexpected end of JSON input"}]}`,
			},
		},
		{
			description: "good - user",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"type":"User","user":{"is_active":true,"locale":"en_US","is_org_admin":false,"username":"jsmith","email":"jsmith@redhat.com","first_name":"John","last_name":"Smith","is_internal":false}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"type":"User","user":{"is_active":true,"locale":"en_US","is_org_admin":false,"username":"jsmith","email":"jsmith@redhat.com","first_name":"John","last_name":"Smith","is_internal":false}}}`,
			},
		},
		{
			description: "good - internal",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"type":"Internal","internal":{"org_id":"1"},"account_number":"1"}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"type":"Internal","account_number":"1","internal":{"org_id":"1"}}}`,
			},
		},
		{
			description: "good - system",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"type":"System","system":{"cn":"a4e67559-1cb5-43e3-bcf7-cb2b0c196bac"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"type":"System","system":{"cn":"a4e67559-1cb5-43e3-bcf7-cb2b0c196bac"}}}`,
			},
		},
		{
			description: "good - associate",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"type":"Associate","associate":{"Role":[],"email":"jsmith@redhat.com","givenName":"John","rhatUUID":"7ce854e8-606e-4d1a-9492-4660fcb4cfa4","surname":"Smith"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"type":"Associate","associate":{"Role":[],"email":"jsmith@redhat.com","givenName":"John","rhatUUID":"7ce854e8-606e-4d1a-9492-4660fcb4cfa4","surname":"Smith"}}}`,
			},
		},
		{
			description: "good - x509",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"type":"X509","x509":{"subject_dn":"abc","issuer_dn":"def"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"type":"X509","x509":{"subject_dn":"abc","issuer_dn":"def"}}}`,
			},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://localhost", nil)
			for k, v := range test.input.headers {
				req.Header.Add(k, v)
			}
			rr := httptest.NewRecorder()
			handler := Identify(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				identity, err := GetIdentity(r)
				if err != nil {
					t.Error(err)
				}
				data, err := json.Marshal(identity)
				if err != nil {
					formatJSONError(w, http.StatusInternalServerError, err.Error())
					return
				}
				if _, err := w.Write(data); err != nil {
					t.Error(err)
				}
			}))
			handler.ServeHTTP(rr, req)
			got := response{rr.Code, rr.Body.String()}

			if !cmp.Equal(got, test.want, cmp.AllowUnexported(response{})) {
				t.Errorf("\ngot: %#v\nwant: %#v", got, test.want)
			}
		})
	}
}

func TetGetIdentity(t *testing.T) {
	tests := []struct {
		description string
		input       *http.Request
		want        *Identity
		wantError   error
	}{
		{
			description: "empty identity",
			input:       httptest.NewRequest(http.MethodGet, "http://localhost", nil).WithContext(context.WithValue(context.Background(), identityKey, &Identity{})),
			want:        &Identity{},
		},
		{
			description: "missing identity",
			input:       httptest.NewRequest(http.MethodGet, "http://localhost", nil),
			wantError:   ErrMissingIdentityValue,
		},
		{
			description: "invalid identity value",
			input:       httptest.NewRequest(http.MethodGet, "http://localhost", nil).WithContext(context.WithValue(context.Background(), identityKey, "")),
			wantError:   TypeCastError{"", Identity{}},
		},
	}

	for _, test := range tests {
		t.Run(test.description, func(t *testing.T) {
			got, err := GetIdentity(test.input)

			if test.wantError != nil {
				if !cmp.Equal(err, test.wantError, cmp.AllowUnexported(TypeCastError{}), cmpopts.EquateErrors()) {
					t.Errorf("%#v != %#v", err, test.wantError)
				}
			} else {
				if err != nil {
					t.Fatal(err)
				}
				if !cmp.Equal(got, test.want) {
					t.Errorf("%v != %v", got, test.want)
				}
			}
		})
	}
}
