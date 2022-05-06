package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/peterbourgon/ff/v3/ffcli"
	"github.com/redhatinsights/module-update-router/identity"
	"github.com/sgreben/flagvar"
)

type allFlags struct {
	Type          string
	AuthType      string
	AccountNumber string
}

type userFlags struct {
	IsActive   bool
	Locale     string
	IsOrgAdmin bool
	Username   string
	Email      string
	FirstName  string
	LastName   string
	IsInternal bool
}

type internalFlags struct {
	OrgID string
}

type systemFlags struct {
	CN string
}

type associateFlags struct {
	Role      flagvar.Strings
	Email     string
	GivenName string
	RHatUUID  string
	Surname   string
}

func main() {
	var (
		allFlags         = allFlags{}
		allFlagSet       = flag.NewFlagSet(filepath.Base(os.Args[0]), flag.ExitOnError)
		userFlags        = userFlags{}
		userFlagSet      = flag.NewFlagSet(fmt.Sprintf("%v %v", filepath.Base(os.Args[0]), "user"), flag.ExitOnError)
		internalFlags    = internalFlags{}
		internalFlagSet  = flag.NewFlagSet(fmt.Sprintf("%v %v", filepath.Base(os.Args[0]), "internal"), flag.ExitOnError)
		systemFlags      = systemFlags{}
		systemFlagSet    = flag.NewFlagSet(fmt.Sprintf("%v %v", filepath.Base(os.Args[0]), "system"), flag.ExitOnError)
		associateFlags   = associateFlags{}
		associateFlagSet = flag.NewFlagSet(fmt.Sprintf("%v %v", filepath.Base(os.Args[0]), "associate"), flag.ExitOnError)
	)

	allFlagSet.StringVar(&allFlags.Type, "type", "", "set the identity.type field to `STRING`")
	allFlagSet.StringVar(&allFlags.AuthType, "auth-type", "", "set the identity.authtype field to `STRING`")
	allFlagSet.StringVar(&allFlags.AccountNumber, "account-number", "111000", "set the identity.account_number field to `NUMBER`")

	userFlagSet.BoolVar(&userFlags.IsActive, "is-active", true, "set the identity.user.is_active field to `BOOL`")
	userFlagSet.StringVar(&userFlags.Locale, "locale", "en_US", "set the identity.user.locale field to `STRING`")
	userFlagSet.BoolVar(&userFlags.IsOrgAdmin, "is-org-admin", false, "set the identity.user.is_org_admin field to `BOOL`")
	userFlagSet.StringVar(&userFlags.Username, "username", "test@redhat.com", "set the identity.user.username field to `STRING`")
	userFlagSet.StringVar(&userFlags.Email, "email", "test@redhat.com", "set the identity.user.email field to `STRING`")
	userFlagSet.StringVar(&userFlags.FirstName, "firstname", "test", "set the identity.user.first_name field to `STRING`")
	userFlagSet.StringVar(&userFlags.LastName, "lastname", "user", "set the identity.user.last_name field to `STRING`")
	userFlagSet.BoolVar(&userFlags.IsInternal, "is-internal", true, "set the identity.user.is_internal field to `BOOL`")

	internalFlagSet.StringVar(&internalFlags.OrgID, "orgid", "10001", "set the identity.internal.org_id field to `STRING`")

	systemFlagSet.StringVar(&systemFlags.CN, "cn", "760e4a9b-c0cc-4538-8b8c-09d1a6335dd2", "set the identity.system.cn field to `STRING`")

	associateFlagSet.Var(&associateFlags.Role, "role", "set the identity.associate.Role field to `STRING` (can be set multiple times)")
	associateFlagSet.StringVar(&associateFlags.Email, "email", "test@redhat.com", "set the identity.associate.email field to `STRING`")
	associateFlagSet.StringVar(&associateFlags.GivenName, "givenname", "test", "set the identity.associate.givenName field to `STRING`")
	associateFlagSet.StringVar(&associateFlags.RHatUUID, "rhatuuid", "204f8e50-40b4-45d2-aa84-4bd7382e94d3", "set the identity.associate.rhatUUID field to `STRING`")
	associateFlagSet.StringVar(&associateFlags.Surname, "surname", "user", "set the identity.associate.surname field to `STRING`")

	root := &ffcli.Command{
		ShortUsage: fmt.Sprintf("%v [flags] <subcommand>", allFlagSet.Name()),
		FlagSet:    allFlagSet,
		Subcommands: []*ffcli.Command{
			{
				Name:       "user",
				ShortUsage: fmt.Sprintf("%v [flags]", userFlagSet.Name()),
				ShortHelp:  "generate a user identity JSON object",
				FlagSet:    userFlagSet,
				Exec: func(ctx context.Context, args []string) error {
					var id identity.Identity

					id.Identity.Type = "User"
					if allFlags.AuthType == "" {
						id.Identity.AuthType = "basic-auth"
					} else {
						id.Identity.AuthType = allFlags.AuthType
					}
					id.Identity.AccountNumber = &allFlags.AccountNumber
					id.Identity.User = newUser()
					id.Identity.User.IsActive = userFlags.IsActive
					id.Identity.User.Locale = userFlags.Locale
					id.Identity.User.IsOrgAdmin = userFlags.IsOrgAdmin
					id.Identity.User.Username = userFlags.Username
					id.Identity.User.Email = userFlags.Email
					id.Identity.User.FirstName = userFlags.FirstName
					id.Identity.User.LastName = userFlags.LastName
					id.Identity.User.IsInternal = userFlags.IsInternal

					data, err := json.Marshal(id)
					if err != nil {
						return fmt.Errorf("cannot marshal data: %w", err)
					}

					fmt.Println(string(data))

					return nil
				},
			},
			{
				Name:       "internal",
				ShortUsage: fmt.Sprintf("%v [flags]", systemFlagSet.Name()),
				ShortHelp:  "generate an internal identity JSON object",
				FlagSet:    internalFlagSet,
				Exec: func(ctx context.Context, args []string) error {
					var id identity.Identity

					id.Identity.Type = "System"
					id.Identity.AuthType = allFlags.AuthType
					id.Identity.AccountNumber = &allFlags.AccountNumber
					id.Identity.Internal = newInternal()
					id.Identity.Internal.OrgID = internalFlags.OrgID

					data, err := json.Marshal(id)
					if err != nil {
						return fmt.Errorf("cannot marshal data: %w", err)
					}

					fmt.Println(string(data))

					return nil
				},
			},
			{
				Name:       "system",
				ShortUsage: fmt.Sprintf("%v [flags]", systemFlagSet.Name()),
				ShortHelp:  "generate a system identity JSON object",
				FlagSet:    systemFlagSet,
				Exec: func(ctx context.Context, args []string) error {
					var id identity.Identity

					id.Identity.Type = "System"
					if allFlags.AuthType == "" {
						id.Identity.AuthType = "cert-auth"
					} else {
						id.Identity.AuthType = allFlags.AuthType
					}
					id.Identity.AccountNumber = &allFlags.AccountNumber
					id.Identity.System = newSystem()
					id.Identity.System.CN = systemFlags.CN

					data, err := json.Marshal(id)
					if err != nil {
						return fmt.Errorf("cannot marshal data: %w", err)
					}

					fmt.Println(string(data))

					return nil
				},
			},
			{
				Name:       "associate",
				ShortUsage: fmt.Sprintf("%v [flags]", systemFlagSet.Name()),
				ShortHelp:  "generate an associate identity JSON object",
				FlagSet:    associateFlagSet,
				Exec: func(ctx context.Context, args []string) error {
					var id identity.Identity

					id.Identity.Type = "User"
					if allFlags.AuthType == "" {
						id.Identity.AuthType = "basic-auth"
					} else {
						id.Identity.AuthType = allFlags.AuthType
					}
					id.Identity.AccountNumber = &allFlags.AccountNumber
					id.Identity.Associate = newAssociate()
					id.Identity.Associate.Role = associateFlags.Role.Values
					id.Identity.Associate.Email = associateFlags.Email
					id.Identity.Associate.GivenName = associateFlags.GivenName
					id.Identity.Associate.RHatUUID = associateFlags.RHatUUID
					id.Identity.Associate.Surname = associateFlags.Surname

					data, err := json.Marshal(id)
					if err != nil {
						return fmt.Errorf("cannot marshal data: %w", err)
					}

					fmt.Println(string(data))

					return nil
				},
			},
		},
		Exec: func(context.Context, []string) error {
			return flag.ErrHelp
		},
	}

	if err := root.ParseAndRun(context.Background(), os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func newUser() *struct {
	IsActive   bool   `json:"is_active"`
	Locale     string `json:"locale"`
	IsOrgAdmin bool   `json:"is_org_admin"`
	Username   string `json:"username"`
	Email      string `json:"email"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	IsInternal bool   `json:"is_internal"`
} {
	return &struct {
		IsActive   bool   `json:"is_active"`
		Locale     string `json:"locale"`
		IsOrgAdmin bool   `json:"is_org_admin"`
		Username   string `json:"username"`
		Email      string `json:"email"`
		FirstName  string `json:"first_name"`
		LastName   string `json:"last_name"`
		IsInternal bool   `json:"is_internal"`
	}{}
}

func newInternal() *struct {
	OrgID string `json:"org_id"`
} {
	return &struct {
		OrgID string `json:"org_id"`
	}{}
}

func newSystem() *struct {
	CN string `json:"cn"`
} {
	return &struct {
		CN string `json:"cn"`
	}{}
}

func newAssociate() *struct {
	Role      []string `json:"Role"`
	Email     string   `json:"email"`
	GivenName string   `json:"givenName"`
	RHatUUID  string   `json:"rhatUUID"`
	Surname   string   `json:"surname"`
} {
	return &struct {
		Role      []string `json:"Role"`
		Email     string   `json:"email"`
		GivenName string   `json:"givenName"`
		RHatUUID  string   `json:"rhatUUID"`
		Surname   string   `json:"surname"`
	}{}
}
