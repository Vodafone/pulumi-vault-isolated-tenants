## About
This is the folder for [Roleset Binding](https://www.vaultproject.io/docs/secrets/gcp#roleset-bindings) definitions.
Read the linked documentation to get a better understanding of what is going on.


## How to enable Vault dynamic Service Account access
1. Grant the Security Admin role to the Vault Service Account (see table below).
1. Create a [Roleset](https://www.vaultproject.io/docs/secrets/gcp#roleset-bindings) in this directory and use it in the code. Feel free to check existing files for reference. The syntax needs to be adapted to Go structures.
1. Deploy this Pulumi stack to apply the Roleset.
1. [Test](https://www.vaultproject.io/docs/secrets/gcp#usage) your configuration.
