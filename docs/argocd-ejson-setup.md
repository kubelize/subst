# ArgoCD CMP with EJSON Configuration

This guide shows how to configure ejson private keys for the ArgoCD Config Management Plugin (CMP).

## Method 1: Kubernetes Secret with Volume Mount (Recommended)

### Step 1: Create the EJSON Secret

```bash
# Create a secret containing your ejson private key
kubectl create secret generic ejson-keys \
  --from-literal=YOUR_PUBLIC_KEY_ID=YOUR_PRIVATE_KEY \
  --namespace argocd

# Example:
kubectl create secret generic ejson-keys \
  --from-literal=5218ea26fa01414883012c8a1c866c5331ebefba069f86a4183090b3b096a771=82d4af0a44dcabe9e44375e2bbe52842ae9497f068eede12833995bc6ab87020 \
  --namespace argocd
```

### Step 2: Update ArgoCD Repo Server Deployment

Add the secret as a volume mount to the CMP sidecar container:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-repo-server
  namespace: argocd
spec:
  template:
    spec:
      containers:
      - name: subst-cmp
        # ... existing CMP sidecar configuration ...
        volumeMounts:
        - name: ejson-keys
          mountPath: /opt/ejson/keys
          readOnly: true
        # ... other volume mounts ...
      volumes:
      - name: ejson-keys
        secret:
          secretName: ejson-keys
      # ... other volumes ...
```

### Step 3: Update CMP Configuration

The CMP plugin will automatically use keys from `/opt/ejson/keys/` directory:

```yaml
# This is already configured in cmp.yaml
spec:
  generate:
    command:
      - /usr/local/bin/subst
    args:
      - render
      - "."
      - --kustomize-build-options
      - "--load-restrictor LoadRestrictionsNone"
      # No --ejson-key needed - automatically found in /opt/ejson/keys/
```

## Method 2: Environment Variable from Secret

### Step 1: Create the Secret

```bash
kubectl create secret generic ejson-config \
  --from-literal=private-key=82d4af0a44dcabe9e44375e2bbe52842ae9497f068eede12833995bc6ab87020 \
  --namespace argocd
```

### Step 2: Update Repo Server with Environment Variable

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-repo-server
  namespace: argocd
spec:
  template:
    spec:
      containers:
      - name: repo-server
        env:
        - name: ARGOCD_ENV_EJSON_KEY
          valueFrom:
            secretKeyRef:
              name: ejson-config
              key: private-key
```

### Step 3: Update CMP to Use Environment Variable

```yaml
spec:
  generate:
    command:
      - /usr/local/bin/subst
    args:
      - render
      - "."
      - --ejson-key
      - "${ARGOCD_ENV_EJSON_KEY}"
      - --kustomize-build-options
      - "--load-restrictor LoadRestrictionsNone"
```

## Method 3: Multiple Keys via ConfigMap + Secret

For multiple ejson keys, combine a ConfigMap for public keys with a Secret for private keys:

### Step 1: Create Public Key ConfigMap

```bash
kubectl create configmap ejson-public-keys \
  --from-literal=public-key-1=5218ea26fa01414883012c8a1c866c5331ebefba069f86a4183090b3b096a771 \
  --from-literal=public-key-2=ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44 \
  --namespace argocd
```

### Step 2: Create Private Keys Secret

```bash
kubectl create secret generic ejson-private-keys \
  --from-literal=5218ea26fa01414883012c8a1c866c5331ebefba069f86a4183090b3b096a771=82d4af0a44dcabe9e44375e2bbe52842ae9497f068eede12833995bc6ab87020 \
  --from-literal=ff4bbf46acd0b467ee48f6e75041bc5b45442bb4b32f4bb0a2bfa928d2c21e44=544f44d4ca525b1a497e39a1e8bb85147749f38d3f38ac25a70940827d0e8c3f \
  --namespace argocd
```

### Step 3: Mount Both in Repo Server

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: argocd-repo-server
  namespace: argocd
spec:
  template:
    spec:
      containers:
      - name: subst-cmp
        # ... existing CMP sidecar configuration ...
        volumeMounts:
        - name: ejson-keys
          mountPath: /opt/ejson/keys
          readOnly: true
      volumes:
      - name: ejson-keys
        secret:
          secretName: ejson-private-keys
```

## Verification

### Test Your Configuration

```bash
# Check if the secret is mounted correctly
kubectl exec -it deployment/argocd-repo-server -n argocd -- ls -la /opt/ejson/keys/

# Test ejson decryption manually
kubectl exec -it deployment/argocd-repo-server -n argocd -- ejson decrypt /path/to/your/encrypted.ejson
```

### Troubleshooting

1. **Keys not found**: Ensure the secret is mounted at `/opt/ejson/keys/`
2. **Permission denied**: Check that the secret files have correct permissions
3. **Decryption fails**: Verify the private key matches the public key in your `.ejson` files

## Security Best Practices

1. **Use RBAC**: Limit access to the ejson secrets
2. **Rotate keys**: Regularly rotate your ejson keys
3. **Audit access**: Monitor who accesses the ejson secrets
4. **Separate environments**: Use different keys for dev/staging/prod

## Example Application Configuration

Your ArgoCD Application should reference repositories with `.ejson` files:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: my-app
  namespace: argocd
spec:
  source:
    repoURL: https://github.com/your-org/your-repo
    path: k8s/overlays/production
    plugin:
      name: subst
  # ... rest of application config
```

The CMP will automatically detect `subst.yaml` files and decrypt any `.ejson` files using the mounted keys.
