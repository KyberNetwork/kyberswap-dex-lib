#!/usr/bin/env python3
# SPDX-License-Identifier: UNLICENSED
# Copyright (c) 2025-2026 Everlong Labs Limited
"""Derive test/kyber/CollRebalancerMathCopy.sol from CollRebalancerMath @ c104 80abd43.

Run from the c104 checkout root. The only edits are: private -> internal on functions and constants, the
LevCurve struct inlined (so the copy does not pull src/hooks/** into the compile), frozenParams() dropped (it is
checked against the deployed library directly) and the library renamed.
"""
import re

src = open("src/hooks/everlong/lev/CollRebalancerMath.sol").read()
types = open("src/hooks/everlong/lev/LevCurveTypes.sol").read()
lev = re.search(r"struct LevCurve \{.*?\n\}", types, re.S).group(0)
out = src.replace('import {LevCurve, LevCurveParams} from "src/hooks/everlong/lev/LevCurveTypes.sol";\n', "")
out = out.replace("library CollRebalancerMath {", lev + "\n\nlibrary CollRebalancerMathCopy {")
fp = re.search(r"    function frozenParams\(\) public pure returns \(LevCurveParams memory p\) \{.*?\n    \}\n", out, re.S)
out = out.replace(fp.group(0), "")
out = out.replace(" private ", " internal ").replace(" private\n", " internal\n").replace("private pure", "internal pure")
out = out.replace("a second, internal copy of these", "a second, private copy of these")
out = out.replace(
    "// SPDX-License-Identifier: UNLICENSED",
    "// SPDX-License-Identifier: UNLICENSED\n// LevCurveEdges: verbatim CollRebalancerMath @ c104 80abd43 with private -> internal (functions and constants),\n"
    "// the LevCurve struct inlined, frozenParams() dropped and the library renamed. Nothing else differs.",
)
open("test/kyber/CollRebalancerMathCopy.sol", "w").write(out)
