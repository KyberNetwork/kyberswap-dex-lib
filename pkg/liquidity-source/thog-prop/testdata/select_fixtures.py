#!/usr/bin/env python3
"""Select unchanged contract observations from the full external capture directory.

Usage: python3 testdata/select_fixtures.py /path/to/full/fixtures
No expected quote is calculated by this script.
"""
import gzip
import hashlib
import json
from pathlib import Path
import sys

source = Path(sys.argv[1])
destination = Path(__file__).resolve().parent
# Both directions: same-category stablecoins, cross-category ordinary tokens,
# ordinary/xStock, and xStock/xStock. The one-block corpus covers all live pairs.
pairs = {(0, 1), (1, 0), (0, 7), (7, 0), (0, 8), (8, 0), (8, 9), (9, 8)}
for name in ['deployed-v8', 'boundaries-v8', 'execution-model3', 'deployed-model3', 'one-block-model3']:
    path = source / (name + '.json.gz')
    original = path.read_bytes()
    fixture = json.loads(gzip.decompress(original))
    total = sum(len(s['cases']) for s in fixture['snapshots'])
    for snapshot in fixture['snapshots']:
        cases = snapshot['cases']
        chosen = {i for i, c in enumerate(cases)
                  if name == 'one-block-model3' or (c['inputIndex'], c['outputIndex']) in pairs}
        if name == 'execution-model3':
            # Preserve the largest measured successful Sell in each category class.
            raw = bytes.fromhex(snapshot['snapshot'][2:])
            word = lambda at: int.from_bytes(raw[at:at+32], 'big')
            base = word(0)
            start = base + word(base + 7*32)
            categories = [word(start + 32 + i*480 + 64) for i in range(word(start))]
            for same in [False, True]:
                candidates = [i for i, c in enumerate(cases)
                              if c['success'] and c['method'] == 'execution'
                              and (categories[c['inputIndex']] == categories[c['outputIndex']]) == same]
                chosen.add(max(candidates, key=lambda i: cases[i]['measuredGas']))
        indices = sorted(chosen)
        snapshot['sourceCaseIndices'] = indices
        snapshot['cases'] = [cases[i] for i in indices]
    fixture['selection'] = {'sourceFile': path.name, 'sourceSha256': hashlib.sha256(original).hexdigest(),
                            'sourceCases': total, 'selector': 'select_fixtures.py'}
    output = destination / path.name
    output.write_bytes(gzip.compress((json.dumps(fixture, separators=(',', ':'))+'\n').encode(), mtime=0))
    print(path.name, sum(len(s['cases']) for s in fixture['snapshots']), 'of', total, 'cases;', output.stat().st_size, 'bytes')
