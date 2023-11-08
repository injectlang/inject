# Injector

Allows you to store config and encrypted secrets in git.  When your container (or service if on a VM) starts,
`injector` (or `injectord` if running a sidecar container) will decrypt secrets and inject the resulting
configs as environment variables before starting your container app (like your API).

## Goals

1. Store config in git, including encrypted secrets.

2. Config versioned with code. 
   Provides a way to "couple" config to a particular version of code.  This provides
   a way to require a new config in code and populate the config values that will be
   needed by that new code.  For example, if I am currently running v1 of code in production,
   and I want to add a new config named `DB_PORT` which specifies the port of the database
   server, I want to be able to write new code to require `DB_PORT` as of v2.  I'll also
   populate `DB_PORT` for dev, staging and production in the `config.inj.hcl` file.

3. Limit who can see production secrets.
   Provides a way to store encrypted secrets in git.  But we don't
   necessarily want all developers to be able to see the decrypted secrets
   in git.  There may be a workflow where a security engineer can see production
   secrets, but not normal developers that work in the git repo regularly.
   If we have three contexts (dev, staging, production), maybe the developers
   can see dev and staging, but not production.  Or maybe they can only see dev.

4. Only dependency is compute (containers or VMs).
   No config "service" needed, like a config database such as consul, an object store like S3 or GCS, or  
   secrets service like Vault or AWS Secrets Manager.  (Well, you should probably store the private key in a secrets 
   service)

## Deployment

You can deploy this in two different ways: sidecar or entrypoint.  If using a VM, the entrypoint pattern applies to
you as well, as you'll be running `injector` which will then fork/exec your daemon.

### entrypoint

TODO

### sidecar

If you opt for a sidecar, you'll have two containers in play:

```
+------------------+               +------------------+
| code container   |               | config container |
|                  |--http-calls-->|                  |
| runs your daemon |               | runs injectd     |
+------------------+               +------------------+
```

Code container is the normal container you would deploy that has _your_ code in it,
For example, your API code that will listen on a port for customer traffic.

config container is a new sidecar container that is responsible for providing
configs (including decrypted secrets) at runtime.

#### Build time

In your Dockerfile, you build your code container like you normally would.
 You add a new config container that looks something like this:

```Dockerfile
FROM config-container AS api-config
ADD config.inj.hcl /
```

You build both code and config containers and tag them with the same version.

Just to provide an example of what a `config.inj.hcl` file might look like, consider this:

```hcl
context "dev" {
  exports = {
    DB_PORT = "83306"
    DB_USERNAME = "db"
    DB_PASSWORD = "db"
  }
}

context "staging" {
  exports = {
    DB_PORT = "3306"
    DB_USERNAME = decrypt("STAGING2022", "AQBdqk2YG3bpPWfQ+hTTBWpaXRE0Tje1c5ZcybCrb0JiAZjDFUazTCNK934evdMrs1GE/ILsrQ==")
    DB_PASSWORD = decrypt("STAGING2022", "AQBdqk3K714Gy4NSXVmwMx95VdTiJ7bs4W0MVAtFYX3udi+YCxRPlwLv7BZymvjqpV4kCkDvHw==")
  }
}

context "production" {
   exports = {
      DB_PORT = "3306"
      DB_USERNAME = decrypt("PROD2023", "AQBdqk2TV5GYxdTeu8SrfL0gfH79Ssefk60sVl2b8zmcuOLXbe5gljCE4Rz5oDdvc/lVigExZ4yw")
      DB_PASSWORD = decrypt("PROD2023", "AQBdqk1SXCc0Yd26m1+XVhs1LrMrhFTf473Hv7bbTS9getqBAYFkSUwRjt0FqcFHyRQMLTwLXVP+I/oWIOczpSqFMg==")
   }
}
```

Here, we have unencrypted configs for the dev context.  Staging has two configs which are
encrypted using asymmetric encryption.  Same with production, but the encrypted values are
different.  The asymmetric keypair used for to encrypt staging configs is different from
the keypair used for production.

#### Run time

We tell our container runtime system (Kubernetes, AWS ECS, etc.) to run the same tag on
both our code container and our config container.  We also set a dependency that the config
container has to be healthy before the code container can start.

The config container needs two context variables to function:

- `CONTEXT_NAME`, which maps to `dev`, `staging` or `production` in above examples
- `PRIVATE_JSON_KEYSET_<KEYNAME>`, (e.g. `PRIVATE_JSON_KEYSET_PROD2022`) which is the private side of the keypair used to
encrypt the secrets above.  

NOTE: to support key rotation, multiple private keys can be used at the same time.  If your context
block were to use multiple keypairs, you must provide multiple environment variables.  

For example, assume you had this `config.inj.hcl` file:
```hcl
context "production" {
   exports = {
      DB_PORT = "3306"
      DB_USERNAME = decrypt("PROD2022", "AQBdqk2TV5GYxdTeu8SrfL0gfH79Ssefk60sVl2b8zmcuOLXbe5gljCE4Rz5oDdvc/lVigExZ4yw")
      DB_PASSWORD = decrypt("PROD2023", "AQBdqk1SXCc0Yd26m1+XVhs1LrMrhFTf473Hv7bbTS9getqBAYFkSUwRjt0FqcFHyRQMLTwLXVP+I/oWIOczpSqFMg==")
   }
}
```
you need to provide both `PRIVATE_JSON_KEYSET_PROD2022` and `PRIVATE_JSON_KEYSET_PROD2023`.


The config container will then load the file `/config.inj.hcl` and decrypt the secrets using
`PRIVATE_JSON_KEYSET_<KEYNAME>`.  If it can't find an context defined in `config.inj.hcl` named `CONTEXT_NAME`
or if the decryption fails, the config container will exit, which should prevent the code
container from coming up.  Assuming we were trying to upgrade from v1 to v2, this should
cause the deploy to fail in Kubernetes/ECS/etc. and a rollback to v1 should occur.

Once the config container is up, the code container comes up.  As the entrypoint, the 
code container contacts the config container via http to get the configs needed.  The code
container loads them into the bash environment, then starts the app.

# Utilities

A few utility programs are provided to do routine tasks.

## `add_pubkey`

Adds a public key to config.inj.hcl file.  After you've generated a public/private keypair with `tinkey`, you want to store the public side 
in the `config.inj.hcl` file, mainly so that other utilities can encrypt secrets using this public key.

```bash
$ ./build/add_pubkey -h
usage: ./build/add_pubkey <key_name> <path_to_public_keyset_json>

Given a HPKE public keyset file (e.g. public_keyset.json), add to config.hcl

To use this program, you'll need the public key from a public/private keypair generated by Tink.
Install the program tinkey, then run:

tinkey create-keyset --key-template=DHKEM_X25519_HKDF_SHA256_HKDF_SHA256_AES_256_GCM --out private_keyset.json
tinkey create-public-keyset --in private_keyset.json --out public_keyset.json

For more info, see https://developers.google.com/tink/exchange-data

example:
./build/add_pubkey PROD2022 public_keyset.json

would base64 encode the public keyset and add it to config.hcl:

public_key "PROD2022" {
    base64 = <<-EOT
      eyJXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX
      XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX==
    EOT
}
```

## `injectd`

TODO

## `injector`

TODO

## `renderinj`

TODO

## `encrypt_string`

Program that can encrypt a string using a `tinkey` generated `public_keyset.json`
This isn't normally needed, but is useful at times for debugging.

```bash
$ ./build/encrypt_string
usage: ./build/encrypt_string '<string_to_encrypt>' '<public_json_keyset>'

Encrypt a string using a public key.  This uses Google Tink's "exchange" method,
which is HPKE (Hybrid Public Key Encryption).  See RFC 9180 for more info on HPKE.

To use this program, you'll need a public/private keypair generated by Tink.
Install the program tinkey, then run:

tinkey create-keyset --key-template=DHKEM_X25519_HKDF_SHA256_HKDF_SHA256_AES_256_GCM --out private_keyset.json
tinkey create-public-keyset --in private_keyset.json --out public_keyset.json

For more info, see https://developers.google.com/tink/exchange-data

example:
./build/encrypt_string "Hello World" "$(cat public_keyset.json)"
 or
./build/encrypt_string "Hello World" '{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePublicKey","value":"EgYIARABGAIaIJO9jY9uaVIcwcl4EuKYjRaCJIsVKdGoC6akdYTP1VxK","keyMaterialType":"ASYMMETRIC_PUBLIC"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}'

would output an encrypted string like (your output will be different, but you can still decrypt the example below):

AQBdqk3v3MinOx72ZUTmRLsMn3KlmU2UUmy+eyzrR03Y4397IExQbNyGzisR0uOoS87CNcH9A1pwfmf/R2fVpQ==
```

## `decrypt_string`

Program that can decrypt a string using a `tinkey` generated `private_keyset.json`
This isn't normally needed, but is useful at times for debugging.

```bash
$ ./build/decrypt_string
usage: ./build/decrypt_string '<string_to_decrypt>' '<private_json_keyset>'

Decrypt a string using a private key (using Google Tink's "exchange" method)

examples:
./build/decrypt_string 'AQBdqk3v3MinOx72ZUTmRLsMn3KlmU2UUmy+eyzrR03Y4397IExQbNyGzisR0uOoS87CNcH9A1pwfmf/R2fVpQ==' '{"primaryKeyId":6138445,"key":[{"keyData":{"typeUrl":"type.googleapis.com/google.crypto.tink.HpkePrivateKey","value":"EioSBggBEAEYAhogk72Nj25pUhzByXgS4piNFoIkixUp0agLpqR1hM/VXEoaIF/bNmedQsiXENLP2shPjEutFFHYtKY1v1CvxrifPpK7","keyMaterialType":"ASYMMETRIC_PRIVATE"},"status":"ENABLED","keyId":6138445,"outputPrefixType":"TINK"}]}'

would output the decrypted string:

Hello World
```


# Inject files

Inject files are a written in injectlang, which really uses [Hashicorp Config Language](https://github.com/hashicorp/hcl) (HCL)
with some required syntax.  Calling it injectlang makes it easier to communicate that you are talking
about a specific dialect of HCL.

## Inject Block Types

### `custom_function`

A custom_function block defines a bash script that you want to be able to run to grab external data in an inject file.
 
#### example

This is an example that grabs a AWS Cloudformation output.

(this is simplified; in real life, you would want to only make the AWS API call `describe-stacks` once and cache it
for all invocations of `cfnOutput`)

NOTE: if you want to use the bash variable interpolation syntax `${}`, you need to escape it and use `$${}` instead.
You see this in the example below with `$${ds}`.

The reason for this is Inject/HCL uses `${}` for variable interpolation.  Alternately, you may be able to use the bash
variable interpolation syntax that omits `{}`, so use `$NAME` instead of `${NAME}`.
```hcl
custom_function "cfnOutput" {
  params  = [region, stack_name, output_name]
  command = <<-EOT
       ds="$(aws --region ${region} cloudformation describe-stacks --stack-name ${stack_name})"
       if [[ $? != 0 ]]; then
         echo "Could not describe stack ${stack_name}: $ds"
         exit 1
       fi
       echo "$${ds}" |
       jq -re '.Stacks[] |
               select(.StackName == "${stack_name}") |
               .Outputs[] |
               select(.OutputKey == "${output_name}") |
               .OutputValue' ||
       (echo "Could not find CFN output ${output_name} in stack ${stack_name}"
        exit 1)
    EOT
}
```

Later in your inject file, you could use this to lookup a Cloudformation output at startup:
```hcl
context "PROD2022" {
  exports = {
    S3_BUCKET_ARN = cfnOutput("us-west-2", "prod", "S3BucketArn")
  }
}

```

### `public_key`

A public_key block stores a Google Tink public key.  You can have more than one public_key block in a inject file.
Public keys are used to encrypt secrets.  To create a new public and private keypair, from the shell, use the command
`add_pubkey`.

#### example

```hcl
public_key "DEV2022" {
  base64 = <<-EOT
      eyJwcmltYXJ5S2V5SWQiOjYxMzg0NDUsImtleSI6W3sia2V5R
      GF0YSI6eyJ0eXBlVXJsIjoidHlwZS5nb29nbGVhcGlzLmNvbS
      9nb29nbGUuY3J5cHRvLnRpbmsuSHBrZVB1YmxpY0tleSIsInZ
      hbHVlIjoiRWdZSUFSQUJHQUlhSUpPOWpZOXVhVkljd2NsNEV1
      S1lqUmFDSklzVktkR29DNmFrZFlUUDFWeEsiLCJrZXlNYXRlc
      mlhbFR5cGUiOiJBU1lNTUVUUklDX1BVQkxJQyJ9LCJzdGF0dX
      MiOiJFTkFCTEVEIiwia2V5SWQiOjYxMzg0NDUsIm91dHB1dFB
      yZWZpeFR5cGUiOiJUSU5LIn1dfQo=
  EOT
}
```

### `context`

An `context` block describes config per "context" where context refers to the
different places you run your compute.  For example, you may normally refer to this as a Cloudformation "stack" or 
maybe an "environment" or a "datacenter".
Example names of contexts might be "dev", "test", "staging", "production".

There is an object in each context called `exports`.  This refers to environment variables that will be injected 
at runtime just before your app is executed in a container or on a VM.

#### example

```hcl
context "production" {
  exports = {
    DB_USERNAME = "db"
    DB_PASSWORD = decrypt("PROD2022", "cGFzc3dvcmQK")
    DB_PORT = "3306"
    S3_BUCKET_ARN = cfnOutput("us-west-2", "prod", "S3BucketArn")
  }
}
```
