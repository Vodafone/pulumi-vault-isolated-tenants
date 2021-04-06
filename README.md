# pulumi-vault-isolated-tenants 

This repository contains a Pulumi program that provisions any Vault instances to be securely used by multiple tenants.

## Motivation

Vault Enterprise has a feature called Namespaces. It allows you to create multiple Vault instances on the same server that give you fully separated environments.

We wanted to replicate this behavior with Vault Community edition.


## Solution

We created this Pulumi program to allow easy separation of tenats with Vault policies.

It introduces the abstraction of Tenant Prefix. It is capable of creating:
* generic key-value prefix for secrets
* policies for accessing the whole prefix (custom templates supported)
* GCP IAM based authentication config for service accounts (with policy mapping)
* GitHub teams access (with policy mapping)

Inside the `config` folder you can see a list of example Prefixes.

## Disclaimer

This abstraction is adapted for Vodafone's use-case - GCP-based access with GitHub authentication.
If you need any other authentication scheme feel free to for the repository.

## Setup

```
pulumi vault login "$your_GCS_bucket"
pulumi stack select production

# set the GCP KMS key to be used for secrets encryption
pulumi stack change-secrets-provider gcpkms://projects/your_key...

# your vault instance authentication
pulumi config set vault:address https://your.vault.url.com/
pulumi config set --secret vault:token "$your_vault_token"

pulumi up
```

## License and Trademarks

git © 2021 Vodafone

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except
in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the 
License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, 
either express or implied. See the License for the specific language governing permissions 
and limitations under the License.

Cask is a trademark of Cask Data, Inc. All rights reserved.

Apache, Apache HBase, and HBase are trademarks of The Apache Software Foundation. Used with
permission. No endorsement by The Apache Software Foundation is implied by the use of these marks.  

