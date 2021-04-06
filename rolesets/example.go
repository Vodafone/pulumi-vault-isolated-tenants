package rolesets

import (
	"vault/tenant"

	"github.com/pulumi/pulumi-vault/sdk/v2/go/vault/gcp"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
)

// ExampleTenantBindings in a form of SecretRolesetBindingArray
var ExampleTenantBindings = []tenant.Roleset{
	{
		RolesetName: "test",
		RolesetBindings: gcp.SecretRolesetBindingArray{
			gcp.SecretRolesetBindingArgs{
				Resource: pulumi.String("buckets/example-bucket"),
				Roles: pulumi.StringArray{
					pulumi.String("roles/storage.objectAdmin"),
					pulumi.String("roles/storage.admin"),
				},
			},
		},
	},
}
