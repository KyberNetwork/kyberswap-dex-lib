#!/usr/bin/env python3
"""Rebuild FlammSampler (abi/) and the test pool MaxOfTwoPool (testdata/) with solc 0.8.35; --check verifies the
committed artifacts. Both are runtime code: a node runs the sampler by a state override, and the tests set the pool's
code at an address."""
import argparse
import json
import pathlib
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--solc', default='solc')
parser.add_argument('--check', action='store_true')
args = parser.parse_args()
version = subprocess.check_output([args.solc, '--version'], text=True)
assert 'Version: 0.8.35+commit.47b9dedd' in version, version
package = pathlib.Path(__file__).resolve().parent.parent
settings = {'viaIR': True, 'optimizer': {'enabled': True, 'runs': 1}, 'evmVersion': 'paris',
            'metadata': {'bytecodeHash': 'none', 'appendCBOR': False},
            'outputSelection': {'*': {'*': ['abi', 'evm.deployedBytecode.object']}}}
outputs = {}
for source, with_abi in ((package / 'abi/FlammSampler.sol', True), (package / 'testdata/MaxOfTwoPool.sol', False)):
    request = {'language': 'Solidity', 'sources': {source.name: {'content': source.read_text()}}, 'settings': settings}
    process = subprocess.run([args.solc, '--standard-json'], input=json.dumps(request), text=True, capture_output=True,
                             check=True)
    result = json.loads(process.stdout)
    for error in result.get('errors', []):
        print(error['formattedMessage'])
    assert not any(error['severity'] == 'error' for error in result.get('errors', []))
    contract = result['contracts'][source.name][source.stem]
    outputs[source.with_suffix('.bin')] = contract['evm']['deployedBytecode']['object'] + '\n'
    if with_abi:
        outputs[source.with_suffix('.json')] = json.dumps(contract['abi'], indent=2) + '\n'
for path, content in outputs.items():
    if args.check:
        assert path.read_text() == content, f'{path.name} is stale; run without --check'
    else:
        path.write_text(content)
print('Sampler artifacts verified' if args.check else 'Sampler artifacts rebuilt')
