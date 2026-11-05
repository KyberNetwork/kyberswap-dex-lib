#!/usr/bin/env python3
"""Rebuild ThogAMM's deployless lens with solc 0.8.25; --check verifies committed artifacts."""
import argparse
import json
import pathlib
import subprocess

parser = argparse.ArgumentParser(description=__doc__)
parser.add_argument('--solc', default='solc')
parser.add_argument('--check', action='store_true')
args = parser.parse_args()
version = subprocess.check_output([args.solc, '--version'], text=True)
assert 'Version: 0.8.25+' in version, version
source = pathlib.Path(__file__).resolve().parent.parent / 'abi/ThogSnapshotLens.sol'
settings = {'viaIR': True, 'optimizer': {'enabled': True, 'runs': 200}, 'evmVersion': 'shanghai',
            'metadata': {'bytecodeHash': 'none', 'appendCBOR': False},
            'outputSelection': {'*': {'*': ['abi', 'evm.bytecode.object']}}}
request = {'language': 'Solidity', 'sources': {source.name: {'content': source.read_text()}}, 'settings': settings}
process = subprocess.run([args.solc, '--standard-json'], input=json.dumps(request), text=True, capture_output=True, check=True)
result = json.loads(process.stdout)
for error in result.get('errors', []):
    print(error['formattedMessage'])
assert not any(error['severity'] == 'error' for error in result.get('errors', []))
contract = result['contracts'][source.name]['ThogSnapshotLens']
outputs = {source.with_suffix('.bin'): contract['evm']['bytecode']['object'] + '\n',
           source.with_suffix('.json'): json.dumps(contract['abi'], indent=2) + '\n'}
for path, content in outputs.items():
    if args.check:
        assert path.read_text() == content, f'{path.name} is stale; run without --check'
    else:
        path.write_text(content)
print('Lens artifacts verified' if args.check else 'Lens artifacts rebuilt')
