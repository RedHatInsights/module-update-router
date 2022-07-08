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
		AccountNumber         *string    `json:"account_number,omitempty"`
		Associate             *Associate `json:"associate,omitempty"`
		AuthType              string     `json:"auth_type,omitempty"`
		EmployeeAccountNumber *string    `json:"employee_account_number,omitempty"`
		Internal              *Internal  `json:"internal,omitempty"`
		OrgID                 string     `json:"org_id"`
		System                *System    `json:"system,omitempty"`
		Type                  *string    `json:"type,omitempty"`
		User                  *User      `json:"user,omitempty"`
		X509                  *X509      `json:"x509,omitempty"`
	} `json:"identity"`
}

// Associate is an embedded data structure for associate-type identifications.
type Associate struct {
	Email     string   `json:"email"`
	GivenName string   `json:"givenName"`
	RHatUUID  string   `json:"rhatUUID"`
	Role      []string `json:"Role"`
	Surname   string   `json:"surname"`
}

// Internal is an embedded data structure for internal-type identifications.
type Internal struct {
	AuthTime    *float64 `json:"auth_time,omitempty"`
	CrossAccess *bool    `json:"cross_access,omitempty"`
	OrgID       string   `json:"org_id"`
}

// System is an embedded data structure for system-type identifications.
type System struct {
	CertType  *string `json:"cert_type,omitempty"`
	ClusterID *string `json:"cluster_id,omitempty"`
	CN        string  `json:"cn"`
}

// User is an embedded data structure for user-type identifications.
type User struct {
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	IsActive   bool   `json:"is_active"`
	IsInternal bool   `json:"is_internal"`
	IsOrgAdmin bool   `json:"is_org_admin"`
	LastName   string `json:"last_name"`
	Locale     string `json:"locale"`
	UserID     string `json:"user_id"`
	Username   string `json:"username"`
}

// X509 is an embedded data structure for x509-type identifications.
type X509 struct {
	SubjectDN string `json:"subject_dn"`
	IssuerDN  string `json:"issuer_dn"`
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
