path "*"
{
  capabilities = ["read", "list"]
}

path "auth/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

path "sys/auth/*"
{
  capabilities = ["create", "update", "delete", "sudo"]
}

path "sys/policies/acl/*"
{
  capabilities = ["read", "list"]
}

path "sys/policies/acl"
{
  capabilities = ["list"]
}

path "*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

path "sys/mounts/*"
{
  capabilities = ["create", "read", "update", "delete", "list", "sudo"]
}

path "sys/health"
{
  capabilities = ["read", "sudo"]
}

path "sys/capabilities"
{
  capabilities = ["create", "update"]
}

path "sys/capabilities-self"
{
  capabilities = ["create", "update"]
}
