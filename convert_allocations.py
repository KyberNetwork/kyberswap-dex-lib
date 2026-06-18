#!/usr/bin/env python3
"""
Script to convert heap allocations to stack allocations in math.go.
Converts new(uint256.Int) and uint256.NewInt() to stack-allocated variables.
"""

import re

def process_file(filepath):
    with open(filepath, 'r') as f:
        content = f.read()
    
    original_content = content
    lines = content.split('\n')
    
    # Track which lines are in package-level var block (lines 12-18)
    # Don't touch these
    
    # Simple heuristic conversions for common patterns
    conversions = []
    
    # Pattern 1: var x = new(uint256.Int).Operation(...)  
    # Replace with: var x uint256.Int; x.Operation(...)
    
    # Pattern 2: x = new(uint256.Int).Operation(...)
    # Replace with: var tmpX uint256.Int; tmpX.Operation(...); x = &tmpX
    
    # Pattern 3: uint256.NewInt(uint64(...))
    # Replace with: var x uint256.Int; x.SetUint64(uint64(...))
    
    # For manual review, let's just create a detailed transformation plan
    print("=== Transformation needed ===")
    print(f"File: {filepath}")
    print(f"Total new(uint256.Int): {content.count('new(uint256.Int)')}")
    print(f"Total uint256.NewInt: {content.count('uint256.NewInt')}")
    
    # Find all function boundaries
    func_pattern = r'^func\s+(\([^)]+\)\s+)?(\w+)\s*\('
    for i, line in enumerate(lines, 1):
        if re.match(func_pattern, line):
            print(f"\nLine {i}: Function start: {line.strip()}")
    
    return content

if __name__ == "__main__":
    import sys
    filepath = "pkg/liquidity-source/syncswapv2/aqua/math.go"
    process_file(filepath)
