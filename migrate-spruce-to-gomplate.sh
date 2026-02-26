#!/bin/bash

# Comprehensive Spruce to Gomplate Migration Script
# Converts testing/new directory from spruce syntax to gomplate syntax

set -e

SEARCH_DIR="testing/new"

echo "========================================"
echo "Spruce to Gomplate Migration Script"
echo "========================================"
echo ""
echo "Target directory: $SEARCH_DIR"
echo ""

# Step 1: Restore from backups if they exist
echo "Step 1: Restoring from backups (if they exist)..."
backup_count=$(find "$SEARCH_DIR" -name "*.bak" 2>/dev/null | wc -l | tr -d ' ')
if [ "$backup_count" -gt 0 ]; then
  find "$SEARCH_DIR" -name "*.bak" | while read -r bakfile; do
    original="${bakfile%.bak}"
    cp "$bakfile" "$original"
  done
  echo "   Restored $backup_count files from backups"
else
  echo "   No backups found, working with current files"
fi
echo ""

# Step 2: Convert spruce syntax to gomplate
echo "Step 2: Converting Spruce syntax to Gomplate..."
file_count=0

while IFS= read -r file; do
  if [ -f "$file" ] && grep -q "((" "$file" 2>/dev/null; then
    # Create backup
    cp "$file" "${file}.bak"
    
    # Apply all conversions in one pass
    perl -i -0777 -pe '
      # Remove (( append )) lines - just delete them, following lines keep their indentation
      s/^\s*-\s*\(\(\s*append\s*\)\)\n//gm;
      
      # Convert grab with default - handle both `: ((` and just `((` at start of value
      # Use single quotes for YAML when default has string quotes  
      s/:\s*\(\(\s*grab\s+\$\.([a-zA-Z0-9_.\-]+)\s+\|\|\s+"([^"]*)"\s*\)\)/: '"'"'\{\{ .$1 | default "$2" \}\}'"'"'/g;
      s/:\s*\(\(\s*grab\s+\$\.([a-zA-Z0-9_.\-]+)\s+\|\|\s+([a-z]+)\s*\)\)/: "\{\{ .$1 | default $2 \}\}"/g;
      
      # Convert simple grab
      s/:\s*\(\(\s*grab\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "{{ .$1 }}"/g;
      
      # Convert concat expressions (multiple patterns) - key: value format
      # Simple 2-part: string + var OR var + string
      s/:\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "$1\{\{ .$2 \}\}"/g;
      s/:\s*\(\(\s*concat\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s*\)\)/: "\{\{ .$1 \}\}$2"/g;
      
      # 3-part: string + var + string
      s/:\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s*\)\)/: "$1\{\{ .$2 \}\}$3"/g;
      
      # 4-part: string + var + string + var
      s/:\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "$1\{\{ .$2 \}\}$3\{\{ .$4 \}\}"/g;
      
      # 5-part: string + var + string + var + string
      s/:\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s*\)\)/: "$1\{\{ .$2 \}\}$3\{\{ .$4 \}\}$5"/g;
      
      # 5-part: var + string + var + string + var
      s/:\s*\(\(\s*concat\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "\{\{ .$1 \}\}$2\{\{ .$3 \}\}$4\{\{ .$5 \}\}"/g;
      
      # 3-part: var + string + var
      s/:\s*\(\(\s*concat\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "\{\{ .$1 \}\}$2\{\{ .$3 \}\}"/g;
      
      # Convert stringify with default empty string
      s/:\s*\(\(\s*stringify\s+\$\.([a-zA-Z0-9_.\-]+)\s+\|\|\s+""\s*\)\)/: "{{ toJSON .$1 | default \\"\\" }}"/g;
      
      # Convert simple stringify
      s/:\s*\(\(\s*stringify\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/: "{{ toJSON .$1 }}"/g;
      
      # Convert list items with concat templates - PRESERVE INDENTATION
      s/^(\s*)-\s*\(\(\s*grab\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/$1- "\{\{ .$2 \}\}"/gm;
      s/^(\s*)-\s*\(\(\s*grab\s+\$\.([a-zA-Z0-9_.\-]+)\s+\|\|\s+"([^"]*)"\s*\)\)/$1- '"'"'\{\{ .$2 | default "$3" \}\}'"'"'/gm;

      # 5-part with 2 strings first: "str1" "str2" $.var1 "str3" $.var2
      s/^(\s*)-\s*\(\(\s*concat\s+"([^"]*)"\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/$1- "$2$3\{\{ .$4 \}\}$5\{\{ .$6 \}\}"/gm;
      
      s/^(\s*)-\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s*\)\)/$1- "$2\{\{ .$3 \}\}$4"/gm;
      s/^(\s*)-\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/$1- "$2\{\{ .$3 \}\}$4\{\{ .$5 \}\}$6\{\{ .$7 \}\}"/gm;
      s/^(\s*)-\s*\(\(\s*concat\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s+\$\.([a-zA-Z0-9_.\-]+)\s+"([^"]*)"\s*\)\)/$1- "$2\{\{ .$3 \}\}$4\{\{ .$5 \}\}$6"/gm;
      s/^(\s*)-\s*\(\(\s*concat\s+"-node-http-proxy="\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/$1- "-node-http-proxy=\{\{ .$2 \}\}"/gm;
      s/^(\s*)-\s*\(\(\s*concat\s+"-node-registry-mirrors="\s+\$\.([a-zA-Z0-9_.\-]+)\s*\)\)/$1- "-node-registry-mirrors=\{\{ .$2 \}\}"/gm;
    ' "$file"
    
    file_count=$((file_count + 1))
    echo "   ✓ Converted: $file"
  fi
done < <(find "$SEARCH_DIR" -type f \( -name "*.yaml" -o -name "*.yml" \))

echo ""
echo "   Total converted: $file_count files"
echo ""

# Step 3: Strip .subst. prefix (subst tool context is already the subst data)
echo "Step 3: Stripping .subst. prefix from converted templates..."
strip_count=0

while IFS= read -r file; do
  if [ -f "$file" ] && grep -q '{{ \.subst\.' "$file" 2>/dev/null; then
    # Replace {{ .subst. with {{ . (preserving original spacing)
    perl -i -pe 's/\{\{(\s*)\.subst\./{{$1./g;' "$file"
    strip_count=$((strip_count + 1))
    echo "   ✓ Stripped: $file"
  fi
done < <(find "$SEARCH_DIR" -type f \( -name "*.yaml" -o -name "*.yml" \))

echo ""
echo "   Total stripped: $strip_count files"
echo ""

# Step 4: Escape Prometheus template variables
echo "Step 4: Escaping Prometheus template variables..."
prom_count=0

while IFS= read -r file; do
  if [ -f "$file" ] && grep -q '\$labels\|\$value\|\$externalLabels' "$file" 2>/dev/null; then
    # Escape {{ $variable }} -> {{`{{ $variable }}`}}
    perl -i -0777 -pe 's/\{\{\s*\$([a-zA-Z0-9_.]+)\s*\}\}/{{`{{ \$$1 }}`}}/g;' "$file"
    prom_count=$((prom_count + 1))
    echo "   ✓ Escaped: $file"
  fi
done < <(find "$SEARCH_DIR" -type f \( -name "*.yaml" -o -name "*.yml" \))

echo ""
echo "   Total escaped: $prom_count files"
echo ""

# Step 5: Verify no spruce syntax remains
echo "Step 5: Verifying conversion..."
remaining=$(grep -r "((\s*grab\|concat\|stringify\|append" "$SEARCH_DIR" --include="*.yaml" 2>/dev/null | wc -l | tr -d ' ')
if [ "$remaining" -gt 0 ]; then
  echo "   ⚠️  Warning: Found $remaining potential unconverted patterns"
  
  # Extract unique files with remaining patterns
  unconverted_files="unconverted-files.txt"
  grep -r "((\s*grab\|concat\|stringify\|append" "$SEARCH_DIR" --include="*.yaml" 2>/dev/null | cut -d: -f1 | sort -u > "$unconverted_files"
  
  unconverted_count=$(wc -l < "$unconverted_files" | tr -d ' ')
  echo "   📝 $unconverted_count files need manual conversion (saved to $unconverted_files)"
  echo ""
  echo "Files requiring manual conversion:"
  while IFS= read -r file; do
    # Count patterns in each file
    pattern_count=$(grep -c "((\s*grab\|concat\|stringify\|append" "$file" 2>/dev/null || echo "0")
    echo "   - $file ($pattern_count pattern$([ "$pattern_count" -ne 1 ] && echo "s"))"
  done < "$unconverted_files"
else
  echo "   ✅ No spruce syntax found"
  rm -f unconverted-files.txt
fi
echo ""

echo "========================================"
echo "Migration Complete!"
echo "========================================"
echo ""
echo "Summary:"
echo "  - Converted $file_count files from Spruce to Gomplate"
echo "  - Stripped .subst. prefix from $strip_count files"
echo "  - Escaped Prometheus templates in $prom_count files"
echo "  - Backups saved with .bak extension"
if [ -f "unconverted-files.txt" ]; then
  unconverted_count=$(wc -l < unconverted-files.txt | tr -d ' ')
  echo "  - $unconverted_count files require manual conversion (see unconverted-files.txt)"
fi
echo ""
echo "To test:"
echo "  ./subst-bin render --kustomize-build-options=\"--load-restrictor=LoadRestrictionsNone\" testing/new/clusters/vclusters/prod/ci-common-prod"
echo ""
if [ -f "unconverted-files.txt" ]; then
  echo "To manually fix remaining patterns:"
  echo "  cat unconverted-files.txt"
  echo ""
fi
echo "To remove backups:"
echo "  find $SEARCH_DIR -name '*.bak' -delete"
echo ""
