package identity

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

var identityKey = struct{}{}

// Identity is the identification data structure set by Cloud Platform 3scale.
type Identity struct {
	Entitlements interface{} `json:"entitlements,omitempty"`
	Identity     struct {
		AccountNumber *string `json:"account_number,omitempty"`
		Associate     *struct {
			Email     string   `json:"email"`
			GivenName string   `json:"givenName"`
			RHatUUID  string   `json:"rhatUUID"`
			Role      []string `json:"Role"`
			Surname   string   `json:"surname"`
		} `json:"associate,omitempty"`
		AuthType              string `json:"auth_type,omitempty"`
		EmployeeAccountNumber string `json:"employee_account_number,omitempty"`
		Internal              *struct {
			AuthTime    float32 `json:"auth_time,omitempty"`
			CrossAccess bool    `json:"cross_access,omitempty"`
			OrgID       string  `json:"org_id"`
		} `json:"internal,omitempty"`
		OrgID  string `json:"org_id"`
		System *struct {
			CertType  string `json:"cert_type,omitempty"`
			ClusterID string `json:"cluster_id,omitempty"`
			CN        string `json:"cn"`
		} `json:"system,omitempty"`
		Type string `json:"type,omitempty"`
		User *struct {
			Email      string `json:"email"`
			FirstName  string `json:"first_name"`
			IsActive   bool   `json:"is_active"`
			IsInternal bool   `json:"is_internal"`
			IsOrgAdmin bool   `json:"is_org_admin"`
			LastName   string `json:"last_name"`
			Locale     string `json:"locale"`
			UserID     string `json:"user_id"`
			Username   string `json:"username"`
		} `json:"user,omitempty"`
		X509 *struct {
			SubjectDN string `json:"subject_dn"`
			IssuerDN  string `json:"issuer_dn"`
		} `json:"x509,omitempty"`
	} `json:"identity"`
}

// Identify returns an http.HandlerFunc that examines the request for the
// presence of the X-Rh-Identity header and adds it to the context.
func Identify(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := r.Header.Get("X-Rh-Identity")
		if data == "" {
			formatJSONError(w, http.StatusBadRequest, "missing X-Rh-Identity header")
			return
		}

		bytes, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			formatJSONError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
			return
		}

		var identity Identity
		if err := json.Unmarshal(bytes, &identity); err != nil {
			formatJSONError(w, http.StatusBadRequest, fmt.Sprintf("%v", err))
			return
		}

		// TODO: One day when the Identity spec is a thing, validate more of it
		// like has non-zero AccoutNumber, Type, etc.

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), identityKey, &identity)))
	})
}

// GetIdentity examines the request context for the Identity value and extracts
// it.
func GetIdentity(r *http.Request) (*Identity, error) {
	v := r.Context().Value(identityKey)
	if v == nil {
		return nil, ErrMissingIdentityValue
	}

	id, ok := v.(*Identity)
	if !ok {
		return nil, TypeCastError{v, Identity{}}
	}

	return id, nil
}
