# Subst Examples

This directory contains clean, focused examples demonstrating the core functionality of subst.

## Examples

### 1. Basic Substitution (`basic-substitution/`)

Demonstrates normal subst.yaml substitutions using gomplate syntax.

**Features:**
- Multiple data sections (`settings`, `environment`)
- Nested value access (`{{ .settings.app.name }}`)
- Simple template processing

**Usage:**
```bash
cd basic-substitution
subst render --kustomize-build-options="--load-restrictor LoadRestrictionsNone"
```

**Template patterns:**
- `{{ .settings.app.name }}` - Access nested configuration
- `{{ .environment.database_url }}` - Access environment variables
- `{{ .settings.cluster.region }}` - Access cluster settings

### 2. EJSON Substitution (`ejson-substitution/`)

Demonstrates ejson decryption and substitution with the `.ejson` namespace.

**Features:**
- Encrypted secrets using EJSON
- Direct access to decrypted values
- Mix of normal substitutions and ejson data

**Usage:**
```bash
cd ejson-substitution
subst render --ejson-key="YOUR_EJSON_KEY" --kustomize-build-options="--load-restrictor LoadRestrictionsNone"
```

**Template patterns:**
- `{{ .ejson.metadata.name }}` - Access ejson Secret metadata
- `{{ index .ejson.data "database-secret" }}` - Access decrypted secret values
- `{{ .settings.app.name }}` - Normal substitutions still work

## Key Concepts

### Data Structure
```yaml
# Available in templates:
settings:          # From subst.yaml
  app: {...}
  cluster: {...}
environment: {...}  # From subst.yaml
ejson:             # From *.ejson files (decrypted)
  apiVersion: v1
  kind: Secret
  metadata: {...}
  data: {...}
```

### Template Syntax
- **Gomplate syntax**: `{{ .path.to.value }}`
- **String values**: Always quoted in output (`'value'`)
- **Index syntax**: Use `{{ index .data "key-with-dashes" }}` for keys with special characters
- **No conflicts**: EJSON data lives under `.ejson` namespace

## Running Examples

Both examples use the `--load-restrictor LoadRestrictionsNone` flag to allow kustomize to access files across directories.

The ejson example requires a valid EJSON private key. The included `app-secret.ejson` uses key: `82d4af0a44dcabe9e44375e2bbe52842ae9497f068eede12833995bc6ab87020`
