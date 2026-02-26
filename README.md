# Subst

[![Go Report Card](https://goreportcard.com/badge/github.com/kubelize/subst)](https://goreportcard.com/report/github.com/kubelize/subst)

A simple extension over kustomize, which allows further variable substitution and introduces simplified yet strong secrets management (for multi tenancy use-cases). Extends the functionality of kustomize for ArgoCD users.

## Functionality

The idea for subst is to act as complementary for kustomize. You can reference additional variables for your environment or from different kustomize paths, which are then accessible across your entire kustomize build. The kustomize you are referencing to is resolved (its paths). In each of these paths you can create new substitution files, which contain variables or secrets, which then can be used by your kustomization. The final output is your built kustomization with the substitutions made.

By default, subst discovers:
- `subst.yaml` files in kustomize paths (for variables)
- `*.ejson` files throughout the directory tree (for encrypted secrets)

## Getting Started

For `subst` to work you must already have a functional kustomize build. Even without any extra substitutions you can run:

```bash
subst render <path-to-kustomize>
```

Which will build the kustomize and process it through gomplate (producing the same output as `kustomize build` if no template variables are used).

### ArgoCD

Install it with the [ArgoCD community chart](https://github.com/argoproj/argo-helm/tree/main/charts/argo-cd). These values should work:

```yaml
...
    repoServer:
      enabled: true
      clusterAdminAccess:
        enabled: true
      containerSecurityContext:
        allowPrivilegeEscalation: false
        capabilities:
          drop:
          - all
        readOnlyRootFilesystem: true
        runAsUser: 1001
        runAsGroup: 1001
      volumes:
      - emptyDir: {}
        name: subst-tmp
      extraContainers:
      - name: cmp-subst
        args: [/var/run/argocd/argocd-cmp-server]
        image: ghcr.io/kubelize/subst-cmp:v1.0.0
        imagePullPolicy: Always
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - all
          readOnlyRootFilesystem: true
          runAsUser: 1001
          runAsGroup: 1001
        resources:
          limits:
            cpu: 500m
            memory: 512Mi
          requests:
            cpu: 100m
            memory: 128Mi
        volumeMounts:
          - mountPath: /var/run/argocd
            name: var-files
          - mountPath: /home/argocd/cmp-server/plugins
            name: plugins
          # Starting with v2.4, do NOT mount the same tmp volume as the repo-server container. The filesystem separation helps
          # mitigate path traversal attacks.
          - mountPath: /tmp
            name: subst-tmp
...
```

Change version accordingly.

**For EJSON private key configuration**, see [ArgoCD EJSON Setup Guide](docs/argocd-ejson-setup.md).

### Paths

The priority is used from the kustomize declaration. First, all the patch paths are read. Then the `resources` are added in given order. So if you want to overwrite something (highest resource), it should be the last entry in the `resources`. The root directory where the kustomization is resolved has the highest priority.

See example `/test/build/kustomization.yaml`

```bash
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization
resources:
  - operators/
  - ../addons/values/high-available
patches:
  - path: ../../apps/common/patches/argo-appproject.yaml
    target:
      kind: AppProject
  - path: ./patches/argo-app-settings.yaml
    target:
      kind: Application
```

Results in the following paths (order by precedence):

  1. /test/build/
  2. /test/build/../addons/values/high-available
  3. /test/build/operators/
  4. /test/build/patches
  5. /test/build/../../apps/common/patches

Note that directories do not resolve by recursion (eg. `/test/build/` only collects files and skips any subdirectories).

### Environment

For environment variables which come from an argo application (`^ARGOCD_ENV_`) we remove the `ARGOCD_ENV_` and they are then available in your substitutions without the `ARGOCD_ENV_` prefix. This way they have the same name you have given them on the application ([Read More](https://argo-cd.readthedocs.io/en/stable/operator-manual/config-management-plugins/#using-environment-variables-in-your-plugin)). All the substitutions are available as flat key, so where needed you can use environment substitution.

## Template Processing

[Gomplate](https://github.com/hairyhenderson/gomplate) is used to process templates. Gomplate provides powerful templating with 100+ built-in functions for string manipulation, encryption, data sources, and more. You can access substitution variables using [Go template syntax](https://docs.gomplate.ca/syntax/).

**Basic example:**
```yaml
name: "{{ .settings.app.name }}"
version: "{{ .settings.app.version }}"
```

**Using gomplate functions:**
```yaml
# String manipulation
upper_name: "{{ .settings.app.name | strings.ToUpper }}"

# Encoding
api_key: "{{ .secrets.key | base64.Encode }}"

# Conditionals
env: "{{ if eq .environment.type "prod" }}production{{ else }}development{{ end }}"
```

See [Gomplate documentation](https://docs.gomplate.ca/) for all available functions and features.

## Secrets

Subst supports [EJSON](https://github.com/Shopify/ejson) for secret decryption. Encrypted `.ejson` files are automatically discovered and decrypted during the build process, with their contents made available under the `.ejson` namespace for template substitution.

### How EJSON Loading Works

1. **File Discovery**: All `.ejson` files in your kustomize directory tree are automatically discovered
2. **Decryption**: Files are decrypted using private keys from disk or CLI flags
3. **Substitution**: Decrypted data is available in templates under `.ejson` namespace

### Private Key Sources

EJSON private keys are loaded from these sources (in order of precedence):

1. **CLI flag**: `--ejson-key` (can be specified multiple times)
   ```bash
   subst render --ejson-key YOUR_PRIVATE_KEY .
   ```

2. **Disk directories** (automatically searched):
   - `/opt/ejson/keys` (for containers)
   - `~/.ejson/keys` (for local usage)

Keys must be named after the public key (hex format) with no file extension.

### Options

**Skip decryption** - Load encrypted files without decrypting them (removes encryption metadata only):
```bash
subst render --skip-decrypt .
```

### EJSON Setup

#### Local Installation

**Go**:
```bash
go install github.com/Shopify/ejson/cmd/ejson@v1.5.3
```

**Brew**: Unfortunately, the [Brew](https://github.com/Shopify/ejson?tab=readme-ov-file#installation) package is unmaintained and not recommended.

#### Creating Encrypted Files

```bash
# Generate a keypair (saves private key to ~/.ejson/keys/)
ejson keygen

# Create an encrypted file
ejson encrypt your-secrets.ejson
```

The encrypted file will contain a `_public_key` field. Subst automatically removes this field after decryption.

## Installation

### Prerequisites

**Gomplate is required** - Subst uses [gomplate](https://github.com/hairyhenderson/gomplate) for template processing.

**Install gomplate:**

```bash
# macOS (Homebrew)
brew install gomplate

# Linux (direct download)
curl -sSL https://github.com/hairyhenderson/gomplate/releases/latest/download/gomplate_linux-amd64 -o /usr/local/bin/gomplate
chmod +x /usr/local/bin/gomplate

# Go
go install github.com/hairyhenderson/gomplate/v4/cmd/gomplate@latest

# Windows (Chocolatey)
choco install gomplate
```

Verify installation:
```bash
gomplate --version
```

**Note:** Docker images (`ghcr.io/kubelize/subst` and `ghcr.io/kubelize/subst-cmp`) include gomplate - no separate installation needed.

### Go

```bash
go install github.com/kubelize/subst/subst@v1.0.0
```

### Docker

```bash
docker run --rm -it ghcr.io/kubelize/subst:v1.0.0 -h
```

### Github releases

[github.com/kubelize/subst/releases](https://github.com/kubelize/subst/releases)
