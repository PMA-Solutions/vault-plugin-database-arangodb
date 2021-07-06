# vault-plugin-database-arangodb

A [Vault](https://www.vaultproject.io/) plugin for [ArangoDB](https://www.arangodb.com/) to generate dynamic database access credentials.

## Build

TODO

## Installation

The Vault plugin system is documented on the [Vault documentation site](https://www.vaultproject.io/docs/internals/plugins.html).

You will need to define a plugin directory using the `plugin_directory` configuration directive, then place the
`vault-plugin-database-arangodb` executable into the directory.

Sample commands for registering and starting to use the plugin:

```bash
$ SHA256=$(shasum -a 256 plugins/vault-plugin-database-arangodb | cut -d' ' -f1)

$ vault secrets enable database

$ vault write sys/plugins/catalog/database/arangodb-database-plugin sha256=$SHA256 \
        command=vault-plugin-database-arangodb
```

Prior to initializing the plugin, ensure that you have created an administration account in ArangoDB. Vault will use the user specified here to create/update/revoke database credentials. That user must have the appropriate permissions to perform actions upon other database users.

## Usage

Plugin initialization:

```bash
$ vault write database/config/arangodb plugin_name="arangodb-database-plugin" \
        connection_url="http://localhost:8529" \
        username="Administrator" \
        password="password" \
        allowed_roles="my_role"
```

### Dynamic Role Creation

Configure a role with the requested collection/database grants:

```bash
$ vault write database/roles/my-role \
        db_name=arangodb \
        creation_statements='{"collection_grants": [{"db": "my-database", "access": "rw"}]}' \
        default_ttl="1m" \
        max_ttl="24h"
```

To retrieve the credentials for the dynamic accounts

```bash
$ vault read database/creds/my-role
Key                Value
---                -----
lease_id           database/creds/my-role/YlgApUA8o7ZxitiqfdzhF8vq
lease_duration     1m
lease_renewable    true
password           078Ee-1D9o4DJirKbFim
username           v-token-my-role-eUcuQXdhoDXjkpv3buTo-1620099446
```

## Developing

You can run `make dev` in the root of the repo to start up a development vault server and automatically register a local build of the plugin. You will need to have a built `vault` binary available in your `$PATH` to do so.
