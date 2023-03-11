# Subst

![subst](./img/subst.png "subst")

__Currently under development and testing.__

A simple extension over kustomize, which allows further variable substitution and introduces simplified yet strong secrets management (for multi tenancy use-cases). Extends to functionality of kustomize for ArgoCD users.

# Functionality




# Available Substitutions

## Precedence


## Spruce

## Environment Substitution




for environment variables which come from an argo application (`^ARGOCD_ENV_`) we remove the `ARGOCD_ENV_` and they are then available in your substitutions without the `ARGOCD_ENV_` prefix. This way they have the same name you have given them on the application ([Read More](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#using-environment-variables-in-your-plugin)). All the substions are available as flat key, so where needed you can use environment substitution


You can disable environment substitution for a single file by giving it an annotation:

```
apiVersion: operatingsystemmanager.k8c.io/v1alpha1
kind: OperatingSystemProfile
metadata:
  name: osp-flatcar
  namespace: kube-system
  annotations:
    kustomize.toolkit.fluxcd.io/substitute: "disabled"
....   
```

Or you can disable environment substition completely:

```
subst render . --envsubst
```

## Secrets

You can both encrypt files which are part of the kustomize build or which are used for substitution. Currently for secret decryption we support both [ejson](https://github.com/Shopify/ejson) and [sops](https://github.com/mozilla/sops). You can use any combination of these decryption providers together. The principal for all decryption provider is, that they should load the private keys while a substiution build is made instead of having a permanent keystore. This allows for secret tenancy (eg. one secret per argo application). 

Decryption can be disabled, in that case the files are just loaded, without their encryption properties (might be useful if you dont have access to the private keys to decrypt the secrets):

```
subst render . --skip-decrypt
```

Decryption can be enforced, if a secret can not be decrypted it's treated as an error:

```
subst render . --must-decrypt
```

### Kubernetes

For all decryptors you can create a kubernetes secret, which contains the private information for secret decryption.



### EJSON





### SOPS

[SOPS](https://github.com/mozilla/sops) is commonly known and also used by [FluxCD](https://fluxcd.io/flux/guides/mozilla-sops/). 


# Running it


## Local installation

**Brew**

```bash
brew tap buttahtoast/subst
```

**Docker**

```bash
docker run -it ghcr.io/buttahtoast/subst -h
```

**Github Releases**

https://github.com/buttahtoast/subst/releases


## ArgoCD Plugin



## CI/CD

TBD








###

EJSON Merger is a simple bash script, which uses the tools [spruce](https://github.com/geofffranks/spruce), [jq](https://stedolan.github.io/jq/) and [ejson](https://github.com/Shopify/ejson). The main purpose of the script is, to merge multiple yaml/yml and json scripts together into one big map, which can be exported to a specific file or stdout. Since the merge process is done via spurce, you are able to use all spruce operators across all your files. See all spruce operators [here](https://github.com/geofffranks/spruce/blob/master/doc/operators.md). This tool gives you great advantages in your CI builds, since you are able to spread information into different files, but can join them with each other. But also, you can safely upload secrets to a repo.


## Testing


Private Key for testing

Public Key:
5218ea26fa01414883012c8a1c866c5331ebefba069f86a4183090b3b096a771
Private Key:
82d4af0a44dcabe9e44375e2bbe52842ae9497f068eede12833995bc6ab87020

And

<pre><code>Public Key:
ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44
Private Key:
b9f24a02dabd1f05c327c51a88f99390dab0835f0e56d4766885648cda2a51d6</pre></code>


## Rules

* Each EJSON file has to contain it's public key. All data is given in the <code>.data</code> field. If this field isn't set, the EJSON file is not considered.
* [JQ](https://stedolan.github.io/jq/) is required
* [Spruce](https://github.com/geofffranks/spruce) is required.
* [EJSON](https://github.com/Shopify/ejson) is optional.
* [Ruby YAML gem](https://ruby-doc.org/stdlib-2.5.1/libdoc/yaml/rdoc/YAML.html) is required for YAML outputs.

## Example

Create a new EJSON Key-Pair ([More Information](https://github.com/Shopify/ejson))

<pre><code>ejson keygen</pre></code>

Will give you something like this (Don't use these!!):

<pre><code>Public Key:
ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44
Private Key:
b9f24a02dabd1f05c327c51a88f99390dab0835f0e56d4766885648cda2a51d6</pre></code>

First create the EJSON File, should look like:

<pre><code>{
  "_public_key": "ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44",
  "data": {
     "some_password": "muchUnsecret",
     "database": {
       "user": "postgresql",
       "password": "postgresql"
     }
  }
}</pre></code>

Next, decrypt the created EJSON file:

<pre><code>$ ejson encrypt example/my-secrets.ejson
Wrote 562 bytes to example/my-secrets.ejson.</pre></code>

The content of <code>example/my-secrets.ejson</code> should look like this now:

<pre><code>{
  "_public_key": "ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44",
  "data": {
     "some_password": "EJ[1:cwva0hMYQ0Si3CVnXPvFOehf9i5Le6IYQXkR8NIYlRc=:aMfnwm79BF02LbN/q9rP6JkjfNVb0RmX:E/aFMFo5YpPIitMqgQYl3DT/POUkhEcKqR2KYQ==]",
     "database": {
       "user": "EJ[1:cwva0hMYQ0Si3CVnXPvFOehf9i5Le6IYQXkR8NIYlRc=:m+rB13UUhxG6k51HuhrIrQsXLJ4g6zJF:/jyC+uV7210F1KzjGHZ8Ub/Eg/EyoF3facU=]",
       "password": "EJ[1:cwva0hMYQ0Si3CVnXPvFOehf9i5Le6IYQXkR8NIYlRc=:hYP7OqZlGkRQc2BjD9bXfUr+8F/otS75:00D6UzYcZKGLeyIBZGii/mNrFw3w7AzW6Ks=]"
     }
  }
}</pre></code>

As you can see your secrets are now encrypted and can only be decrypted by the matching private key. This allows you to store secrets in git repositories in a secure way, if that's what you are looking for. Next we want to create a Reference to these secrets. let's create <code>example/configuration.yml</code>:

<pre><code>...
  # Make Usage of Spruce Operators
  user: (( grab $.SECRETS.database.user ))
  host: postgres
  password: (( grab $.SECRETS.database.user ))
  database: (( concat $.PROJECT.name "-" $.PROJECT.version ))</pre></code>

Now you it's time to implement the decryption and merging part, which means running the script with correct parameters. It's very simple, please consider the Options section for all possibilities. For this example, the script needs the private key and the destination (where ejson and other configs are located [./example]). We want the Output to STDOUT in YAML format. The command would look like this:

<pre><code>bash ejson-merger.sh -p "b9f24a02dabd1f05c327c51a88f99390dab0835f0e56d4766885648cda2a51d6" -m ./example -s ./example -y</pre></code>

If we haven't done any Syntax Errors the output will look like this:

<pre><code>---
PROJECT:
  maintainer: oliverbaehler
  name: example
  version: 1.0.0
database:
  args:
    database: example-1.0.0
    host: postgres
    password: postgresql
    user: postgresql
  name: example
report_stats: false
server_name: localhost:8800
signing_key_path: "/src/.buildkite/test.signing.key"
suppress_key_server_warning: true
trusted_key_servers:
- server_name: matrix.org</pre></code>

As you can see our secrets were merged in clear text and other spruce operations were executed as well. Since giving this information to STDOUT makes the entire encryption part useless, I would suggest you using the <code>-f</code> option. This way the output will be generated into a file which then can be use by other Scripts or whatever. It's time, try it yourself? :)

You can find these files in the [Example Folder](./example)

## Options

To see all options available, call the <code>-h</code> option:

<pre><code>ejson-merger.sh -h</pre></code>

The Script currently supports the following options:

<pre><code>Usage: ejson-merger.sh [-h] [-p] "ejson_key" [-s] "directory" [-m] "merge_directory" [-k] "secret_key" [-f] "dir/filename" [-r]
    -p ejson_key       Add EJSON private key to decrypt ejson files
    -s directory       Source directory for ejson files (will be searched recursive) [Default "."]
    -m merge_dir       Directory where your json/yml files are located to merge with secrets [Default "."]
    -k secret_key      Top level key for secrets to be mapped to - (( grab $.secret_key.* )) [Default "SECRETS"]
    -f dir/filename    Merge all files with Secrets in one json file. Given parameter is the name of the generated JSON/YAML
    -r                 Remove Secrets from merged files [Default: True]
    -y                 Output in YAML format [Default: JSON]
    -h                 Show this context

Script logs events/errors to [./ejson-merger.sh.log]</pre></code>
