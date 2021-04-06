package tenant

import (
	"fmt"

	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault"
	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault/gcp"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi/config"
)

// PrefixArgs is a configuration for namespacedTenant function
type PrefixArgs struct {
	// Name of the Prefix. Will be used for all resource names.
	Name string `yaml:"Name"`
	// List of GCP Project names from which the Service Accounts will be able to call Vault
	GcpProjects []string `yaml:"GcpProjects"`
	// List of Service Account names that will be able to read the secrets
	ServiceAccounts []string `yaml:"ServiceAccounts"`
	// List of GitHub Team names in format "Organization/TeamName". Team members will be granted write access to the Prefix.
	GithubTeamNames []string `yaml:"GithubTeamNames"`
	// List of RolesetBinding definitions. Every RolesetBinding will be available for all Prefix Readers.
	RolesetBindings []Roleset `yaml:"RolesetBindings"`
	// IAP backend flag
	EnableIAMLogin bool `yaml:"EnableIAMLogin"`
	// List of RolesetBinding definitions. Every RolesetBinding will be available for all Prefix Readers.
	AppRoles []string `yaml:"AppRoles"`
	// Custom template path used to generate "Write" policy.
	PolicyTemplatePath string `yaml:"PolicyTemplatePath"`
	// Custom template path used to generate "Read" policy.
	ReadPolicyTemplatePath string `yaml:"ReadPolicyTemplatePath"`
}

// Roleset is a helper structure defining a singe roleset binding with its name
type Roleset struct {
	RolesetName     string
	RolesetBindings gcp.SecretRolesetBindingArray
}

// PrefixComponent is the pulumi component resource
type PrefixComponent struct {
	pulumi.ResourceState
}

// Prefix creates a Group with aliases per each github team in the given GitHub Organization
// It creates a narrow policy for accessing secrets only in this namespace
func Prefix(ctx *pulumi.Context, authBuilder *AuthBuilder, tenant PrefixArgs, opts ...pulumi.ResourceOption) PrefixComponent {
	tenantPrefix := &PrefixComponent{}
	err := ctx.RegisterComponentResource("neuron:vault:TenantPrefix", tenant.Name, tenantPrefix, opts...)
	Check(err)

	writePolicyName := fmt.Sprintf("%s-policy-write", tenant.Name)
	if tenant.PolicyTemplatePath == "" {
		tenant.PolicyTemplatePath = "policies/tenant_write.hcl.gotmpl"
	}
	writeTpl := readTemplate(tenant.PolicyTemplatePath, TemplateConfig{Tenant: tenant})
	writePolicy, err := vault.NewPolicy(ctx, writePolicyName, &vault.PolicyArgs{
		Name:   pulumi.StringPtr(writePolicyName),
		Policy: pulumi.String(writeTpl),
	}, pulumi.Parent(tenantPrefix))
	Check(err)

	readPolicyName := fmt.Sprintf("%s-policy-read", tenant.Name)
	if tenant.ReadPolicyTemplatePath == "" {
		tenant.ReadPolicyTemplatePath = "policies/tenant_read.hcl.gotmpl"
	}
	readTpl := readTemplate(tenant.ReadPolicyTemplatePath, TemplateConfig{Tenant: tenant})
	readPolicy, err := vault.NewPolicy(ctx, readPolicyName, &vault.PolicyArgs{
		Name:   pulumi.StringPtr(readPolicyName),
		Policy: pulumi.String(readTpl),
	}, pulumi.Parent(tenantPrefix))
	Check(err)

	if len(tenant.ServiceAccounts) > 0 {
		accounts := make(pulumi.StringArray, len(tenant.ServiceAccounts))
		for i, acc := range tenant.ServiceAccounts {
			accounts[i] = pulumi.String(acc)
		}

		projects := make(pulumi.StringArray, len(tenant.GcpProjects))
		for i, pr := range tenant.GcpProjects {
			projects[i] = pulumi.String(pr)
		}

		if tenant.EnableIAMLogin {
			_, err = gcp.NewAuthBackendRole(ctx, fmt.Sprintf("%s-iam", tenant.Name), &gcp.AuthBackendRoleArgs{
				Role:                 pulumi.String(fmt.Sprintf("%s-iam", tenant.Name)),
				Type:                 pulumi.String("iam"),
				BoundProjects:        projects,
				BoundServiceAccounts: accounts,
				TokenPolicies:        pulumi.StringArray{pulumi.String(readPolicyName)},
				TokenTtl:             pulumi.Int(1 * 60 * 60), // 1h
				}, pulumi.Parent(tenantPrefix), pulumi.DeleteBeforeReplace(true))
				Check(err)
		}

		_, err = gcp.NewAuthBackendRole(ctx, fmt.Sprintf("%s-gce", tenant.Name), &gcp.AuthBackendRoleArgs{
			Role:                 pulumi.String(fmt.Sprintf("%s-gce", tenant.Name)),
			Type:                 pulumi.String("gce"),
			BoundProjects:        projects,
			BoundServiceAccounts: accounts,
			TokenPolicies:        pulumi.StringArray{pulumi.String(readPolicyName)},
			TokenTtl:             pulumi.Int(1 * 60 * 60), // 1h
		}, pulumi.Parent(tenantPrefix), pulumi.DeleteBeforeReplace(true))
		Check(err)
	}

	for _, teamName := range tenant.GithubTeamNames {
		addGithubTeam(authBuilder, teamName, writePolicy)
	}
	for _, appRoleName := range tenant.AppRoles {
		addAppRole(authBuilder, appRoleName, readPolicy)
	}

	_, err = vault.NewMount(ctx, fmt.Sprintf("%s-kv-engine", tenant.Name), &vault.MountArgs{
		Path:        pulumi.String(fmt.Sprintf("%s/data", tenant.Name)),
		Description: pulumi.String(fmt.Sprintf("Key Value engine for %s.", tenant.Name)),
		Type:        pulumi.String("kv"),
	}, pulumi.Parent(tenantPrefix))

	gcpBackend, err := gcp.NewSecretBackend(ctx, fmt.Sprintf("%s-gcp-dynamic-secrets", tenant.Name), &gcp.SecretBackendArgs{
		Path:                   pulumi.String(fmt.Sprintf("%s/gcp", tenant.Name)),
		DefaultLeaseTtlSeconds: pulumi.Int(15 * 60),     // 15m
		MaxLeaseTtlSeconds:     pulumi.Int(3 * 60 * 60), // 3h
		// Use default GKE Service Account
		// Credentials: ...,
	}, pulumi.Parent(tenantPrefix))

	if tenant.RolesetBindings != nil {
		conf := config.New(ctx, "gcp")
		gcpProject := conf.Require("project")
		for _, binding := range tenant.RolesetBindings {
			tokenName := fmt.Sprintf("%s-gcp-token-%s", tenant.Name, binding.RolesetName)
			_, err = gcp.NewSecretRoleset(ctx, tokenName, &gcp.SecretRolesetArgs{
				Roleset:    pulumi.String(tokenName),
				Backend:    gcpBackend.ID(),
				Bindings:   binding.RolesetBindings,
				SecretType: pulumi.StringPtr("access_token"),
				Project:    pulumi.String(gcpProject),
				TokenScopes: pulumi.StringArray{
					pulumi.String("https://www.googleapis.com/auth/cloud-platform"),
				},
			}, pulumi.Parent(gcpBackend))
			Check(err)

			keyName := fmt.Sprintf("%s-gcp-key-%s", tenant.Name, binding.RolesetName)
			_, err = gcp.NewSecretRoleset(ctx, keyName, &gcp.SecretRolesetArgs{
				Roleset:    pulumi.String(keyName),
				Backend:    gcpBackend.ID(),
				Bindings:   binding.RolesetBindings,
				SecretType: pulumi.StringPtr("service_account_key"),
				Project:    pulumi.String(gcpProject),
			}, pulumi.Parent(gcpBackend))
			Check(err)
		}
	}

	return *tenantPrefix
}

// ServerConfiguration sets up basic Vault server-wide config
func ServerConfiguration(ctx *pulumi.Context) {
	_, err := vault.NewAudit(ctx, "audit", &vault.AuditArgs{
		Type:        pulumi.String("file"),
		Path:        pulumi.String("file"),
		Description: pulumi.String(""),
		Options: pulumi.StringMap{
			"file_path": pulumi.String("stdout"),
		},
	}, pulumi.Protect(false))
	Check(err)
}
