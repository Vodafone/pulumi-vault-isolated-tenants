package main

import (
	"io/ioutil"
	"vault/tenant"

	"github.com/pulumi/pulumi/sdk/v2/go/pulumi"
	"github.com/pulumi/pulumi/sdk/v2/go/pulumi/config"
	"gopkg.in/yaml.v2"
)

func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		tenant.ServerConfiguration(ctx)

		authBuilder := tenant.NewAuthBuilder()
		var tenantPrefixes []tenant.PrefixArgs

		c := config.New(ctx, "stack")
		configPath := c.Require("configPath")
		files, err := ioutil.ReadDir(configPath)
		tenant.Check(err)
		for _, file := range files {
			yamlFile, err := ioutil.ReadFile(configPath + "/" + file.Name())
			tenant.Check(err)
			err = yaml.UnmarshalStrict(yamlFile, &tenantPrefixes)
			tenant.Check(err)

			for _, prefix := range tenantPrefixes {
				tenant.Prefix(ctx, authBuilder, prefix)
			}
		}

		tenant.BuildAuth(ctx, authBuilder)

		return nil
	})
}
