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
			description: "invalid base64",
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
			description: "invalid json",
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
			description: "user",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"org_id":"12345","user":{"email":"jsmith@redhat.com","first_name":"John","is_active":true,"is_internal":true,"is_org_admin":false,"last_name":"Smith","locale":"en_US","user_id":"jsmith","username":"jsmith"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"org_id":"12345","user":{"email":"jsmith@redhat.com","first_name":"John","is_active":true,"is_internal":true,"is_org_admin":false,"last_name":"Smith","locale":"en_US","user_id":"jsmith","username":"jsmith"}}}`,
			},
		},
		{
			description: "internal",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"internal":{"auth_time":1,"cross_access":true,"org_id":"12345"},"org_id":"12345"}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"internal":{"auth_time":1,"cross_access":true,"org_id":"12345"},"org_id":"12345"}}`,
			},
		},
		{
			description: "system",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"org_id":"12345","system":{"cert_type":"consumer","cluster_id":"1","cn":"a4e67559-1cb5-43e3-bcf7-cb2b0c196bac"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"org_id":"12345","system":{"cert_type":"consumer","cluster_id":"1","cn":"a4e67559-1cb5-43e3-bcf7-cb2b0c196bac"}}}`,
			},
		},
		{
			description: "associate",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"associate":{"email":"jsmith@redhat.com","givenName":"John","rhatUUID":"f54b46b8-2c0d-4fbf-af21-f5cd877d715a","Role":["user"],"surname":"Smith"},"org_id":"12345"}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"associate":{"email":"jsmith@redhat.com","givenName":"John","rhatUUID":"f54b46b8-2c0d-4fbf-af21-f5cd877d715a","Role":["user"],"surname":"Smith"},"org_id":"12345"}}`,
			},
		},
		{
			description: "x509",
			input: request{
				headers: map[string]string{
					"X-Rh-Identity": base64.StdEncoding.EncodeToString([]byte(`{"identity":{"org_id":"12345","x509":{"subject_dn":"O = 12345, CN = 2f1d5e64-67ea-40de-a190-83e86ffed03e","issuer_dn":"O = Red Hat"}}}`)),
				},
			},
			want: response{
				code: http.StatusOK,
				body: `{"identity":{"org_id":"12345","x509":{"subject_dn":"O = 12345, CN = 2f1d5e64-67ea-40de-a190-83e86ffed03e","issuer_dn":"O = Red Hat"}}}`,
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
				t.Errorf("\n%v", cmp.Diff(got, test.want, cmp.AllowUnexported(response{})))
			}
		})
	}
}

func TestGetIdentity(t *testing.T) {
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
					t.Errorf("%v", cmp.Diff(got, test.want))
				}
			}
		})
	}
}
