# Spruce to Gomplate Migration Status

## Summary
- **Total patterns found**: ~558
- **Successfully converted**: 508 (91%)  
- **Remaining**: 50 (9% - complex patterns requiring manual conversion)

## Migration Steps

The migration script performs 5 steps:

1. **Restore from backups** - Reverts to .bak files if they exist
2. **Convert Spruce syntax to Gomplate** - Transforms grab, concat, stringify patterns
3. **Strip .subst. prefix** - Removes `.subst.` since gomplate context is already the subst data
4. **Escape Prometheus templates** - Protects `{{ $labels }}` from gomplate evaluation
5. **Verify conversion** - Lists remaining unconverted patterns

### Why Strip .subst. Prefix?

In Spruce, the data structure was accessed as `$.subst.cluster.name` where `$.subst` represented the root of the subst data.

In Gomplate with subst, the root context `.` already IS the subst data, so:
- Spruce: `(( grab $.subst.cluster.name ))` 
- Gomplate: `{{ .cluster.name }}` (not `{{ .subst.cluster.name }}`)

The migration script automatically strips the `.subst.` prefix from all converted templates (306 files).

## Successfully Converted Patterns

### Simple grab
- `(( grab $.subst.path ))` → `"{{ .path }}"`
- `(( grab $.subst.path || "default" ))` → `'{{ .path | default "default" }}'`

### Concat (2-5 parts)
- `(( concat "str" $.subst.var ))` → `"str{{ .var }}"`
- `(( concat $.subst.var "str" ))` → `"{{ .var }}str"`
- `(( concat "str1" $.subst.var1 "str2" $.subst.var2 "str3" ))` → `"str1{{ .var1 }}str2{{ .var2 }}str3"`

### Stringify
- `(( stringify $.subst.var ))` → `"{{ toJSON .var }}"`

### Append Operator  
- `- (( append ))` lines removed (gomplate uses deep merge instead)

### Prometheus Templates
- Escaped 37 files with `{{ $labels.xxx }}` → `{{`{{ $labels.xxx }}`}}`

## Files Requiring Manual Conversion (53 patterns in 33 files)

The migration script generates `unconverted-files.txt` with a complete list.

### Pattern Types Requiring Manual Conversion

1. **Patterns inside quoted strings**:
   ```yaml
   matchName: "(( grab $.subst.settings.proxy.host ))"
   ```
   → Should be: `matchName: "{{ .settings.proxy.host }}"`

2. **Patterns inside YAML arrays**:
   ```yaml
   value: ["default", (( grab $.subst.cluster.name ))]
   ```
   → Should be: `value: ["default", "{{ .cluster.name }}"]`

3. **Complex multi-part concats (6+ parts)**:
1. `testing/new/apps/operators/stable/observability-legacy/grafana/grafana.yaml`
   - Line 96: Complex role_attribute_path with nested quotes and JMESPath syntax
   - Line 100: SMTP host concat

2. `testing/new/apps/operators/stable/features/crossplane/packages/*/application.yaml`
   - JSON credential strings with embedded quotes
   - Accessor patterns with nested grab

### Subst.yaml Configuration Files (30 files)
Files in `testing/new/apps/overlays/segments/*/subst.yaml`:
- These contain variable definitions, may not need conversion if not processed as templates

### Storage/Secret Files (10 files)
Files in `testing/new/apps/addons/storageclasses/nutanix/*/`:
- Base64 encoded strings with concat
- Secret data patterns

## Migration Script

Location: `/Users/dan/Git/kubelize/subst/migrate-spruce-to-gomplate.sh`

### Features
- Restores files from `.bak` backups before running
- Converts 90%+ of patterns automatically
- Escapes Prometheus template variables
- Handles hyphenated paths (e.g., `kubernetes-dashboard`)
- Fixes indentation after removing append operators

### Usage
```bash
./migrate-spruce-to-gomplate.sh
```

### Output
- Creates backups of all modified files (`.bak` extension)
- Generates `unconverted-files.txt` listing all files requiring manual conversion
- Shows pattern count for each unconverted file
- Provides summary with conversion statistics

## Testing Status

### Examples ✅
- `examples/basic-substitution` - WORKING
- `examples/ejson-substitution` - WORKING (tested earlier)

### Production Clusters
- `testing/new/clusters/vclusters/prod/ci-common-prod` - BLOCKED
  - Indentation issue in capsule operator - FIXED
  - Gomplate parse error at line 16661 - investigating
  - Likely caused by unconverted patterns in referenced apps

## Next Steps

1. **Manual Fixes Needed**:
   - Convert complex concat patterns in grafana.yaml
   - Fix crossplane credential JSON patterns
   - Review subst.yaml files to determine if conversion needed

2. **Testing**:
   - Continue debugging ci-common-prod rendering
   - Run performance comparison once rendering works
   - Validate all converted files render correctly

3. **Script Improvements**:
   - Add support for 6+ part concat patterns  
   - Handle JSON strings with embedded quotes
   - Better detection of patterns in subst.yaml vs templates

## Files Modified

### Fixed Indentation
- `testing/new/apps/operators/stable/scheduling/capsule/operator/app.yaml`

### Converted (318 files total)
All YAML files in testing/new directory with spruce syntax

### Escaped Prometheus (37 files)
Alertrule files in testing/new/apps/operators/stable/observability/prometheus/

## Commands

### Check remaining patterns
```bash
grep -r '((\s*grab\|concat\|stringify' testing/new --include='*.yaml' | wc -l
```

### List files needing conversion
```bash
grep -r '((\s*grab\|concat\|stringify' testing/new/apps --include='*.yaml' | cut -d: -f1 | sort -u
```

### Test rendering
```bash
./subst-bin render --kustomize-build-options="--load-restrictor=LoadRestrictionsNone" testing/new/clusters/vclusters/prod/ci-common-prod
```

