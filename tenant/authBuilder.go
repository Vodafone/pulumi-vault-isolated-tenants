package tenant

import (
	"fmt"
	"strings"

	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault"
	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault/appRole"
	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault/gcp"
	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault/github"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

// GithubOrg defines mapping from a team to a list of Vault Policies
type GithubOrg struct {
	Name        string
	GithubTeams map[string]*GithubTeam
}

// GithubTeam defines mapping from a team to a list of Vault Policies
type GithubTeam struct {
	TeamName string
	Policies []vault.Policy
}

// AuthBuilder contains helper structures for dynamically building policy mappings
type AuthBuilder struct {
	GithubOrgs map[string]*GithubOrg
	AppRoles   map[string][]vault.Policy
}

// AuthBackendComponent represents pulumi resource component for auth backends
type AuthBackendComponent struct {
	pulumi.ResourceState
}

// NewAuthBuilder constructor
func NewAuthBuilder() *AuthBuilder {
	var builder AuthBuilder
	builder.GithubOrgs = make(map[string]*GithubOrg, 0)
	builder.AppRoles = make(map[string][]vault.Policy, 0)

	return &builder
}

func addGithubTeam(builder *AuthBuilder, orgTeam string, policy *vault.Policy) {
	chunks := strings.Split(orgTeam, "/")
	organization := chunks[0]
	team := chunks[1]

	_, contains := builder.GithubOrgs[organization]
	if !contains {
		builder.GithubOrgs[organization] = &GithubOrg{
			Name:        organization,
			GithubTeams: make(map[string]*GithubTeam),
		}
	}

	orgEntry := builder.GithubOrgs[organization]

	_, contains = orgEntry.GithubTeams[team]
	if !contains {
		orgEntry.GithubTeams[team] = &GithubTeam{
			TeamName: team,
			Policies: make([]vault.Policy, 0),
		}
	}

	teamEntry := orgEntry.GithubTeams[team]
	teamEntry.Policies = append(teamEntry.Policies, *policy)
}

func addAppRole(builder *AuthBuilder, appRoleName string, policy *vault.Policy) {
	_, contains := builder.AppRoles[appRoleName]
	if !contains {
		builder.AppRoles[appRoleName] = make([]vault.Policy, 0)
	}

	builder.AppRoles[appRoleName] = append(builder.AppRoles[appRoleName], *policy)
}

// BuildAuth generates final Pulumi objects from auth definitions
func BuildAuth(ctx *pulumi.Context, builder *AuthBuilder, opts ...pulumi.ResourceOption) {
	authBackendComponent := &AuthBackendComponent{}
	err := ctx.RegisterComponentResource("neuron:vault:Authentication", "vault-authentication", authBackendComponent, opts...)
	Check(err)

	_, err = gcp.NewAuthBackend(ctx, fmt.Sprintf("auth-gcp"), &gcp.AuthBackendArgs{
		Path: pulumi.StringPtr("gcp"),
	}, pulumi.Parent(authBackendComponent))
	Check(err)

	for _, organization := range builder.GithubOrgs {
		org := githubOrganization(ctx, organization.Name, pulumi.Parent(authBackendComponent))
		for _, team := range organization.GithubTeams {
			policies := make(pulumi.StringArray, len(team.Policies))

			for i, policyObj := range team.Policies {
				policies[i] = policyObj.Name
			}

			_, err := github.NewTeam(ctx, team.TeamName, &github.TeamArgs{
				Backend:  org.Path,
				Team:     pulumi.String(team.TeamName),
				Policies: policies,
			}, pulumi.Parent(org))
			Check(err)
		}
	}

	approle, err := vault.NewAuthBackend(ctx, "approle", &vault.AuthBackendArgs{
		Type: pulumi.String("approle"),
	})

	for appRoleName, policiesObj := range builder.AppRoles {
		policies := make(pulumi.StringArray, len(policiesObj))

		for i, policyObj := range policiesObj {
			policies[i] = policyObj.Name
		}
		authBackendRole, err := appRole.NewAuthBackendRole(ctx, fmt.Sprintf("%s-appRoleBackend", appRoleName), &appRole.AuthBackendRoleArgs{
			Backend:       approle.Path,
			TokenPeriod:   pulumi.IntPtr(8 * 60 * 60), // 8h
			TokenPolicies: policies,
			RoleName:      pulumi.String(appRoleName),
		}, pulumi.Parent(approle))
		Check(err)

		secretID, err := appRole.NewAuthBackendRoleSecretID(ctx, fmt.Sprintf("%s-appRoleSecret", appRoleName), &appRole.AuthBackendRoleSecretIDArgs{
			Backend:  approle.Path,
			RoleName: authBackendRole.RoleName,
		}, pulumi.Parent(authBackendRole))
		Check(err)

		// _,err = appRole.NewAuthBackendLogin(ctx, fmt.Sprintf("%s-appRoleLogin", appRoleName), &appRole.AuthBackendLoginArgs{
		// 	RoleId: authBackendRole.RoleId,
		// 	Backend: approle.Path,
		// }, pulumi.Parent(authBackendRole))
		Check(err)

		ctx.Export(fmt.Sprintf("AppRole-%s-SecretWrappingTokens", appRoleName), secretID.WrappingToken)
		ctx.Export(fmt.Sprintf("AppRole-%s-RoleId", appRoleName), authBackendRole.RoleId)
	}
}

func githubOrganization(ctx *pulumi.Context, organization string, opts ...pulumi.ResourceOption) *github.AuthBackend {
	githubBackend, err := github.NewAuthBackend(ctx, fmt.Sprintf("%s-auth-github", organization), &github.AuthBackendArgs{
		Path:         pulumi.StringPtr(organization),
		BaseUrl:      pulumi.StringPtr("https://github.vodafone.com/api/v3/"),
		Organization: pulumi.String(organization),
		Tune: github.AuthBackendTuneArgs{
			DefaultLeaseTtl:   pulumi.StringPtr("8h"),
			MaxLeaseTtl:       pulumi.StringPtr("24h"),
			TokenType:         pulumi.StringPtr("default-service"),
			ListingVisibility: pulumi.StringPtr("unauth"),
		},
	}, opts...)
	Check(err)

	ctx.Export(fmt.Sprintf("github_%s_accessor", organization), githubBackend.Accessor)

	return githubBackend
}
