#!/usr/bin/env python3
"""Capture the stored-multiplier lens and prove issuer getters are not consulted.

All calls are read-only. The isolation case overrides only token code inside
eth_call: sharesOf returns the observed shares, and every other selector reverts.
"""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
import subprocess
from refresh_probe import CODE, PROXY, checked, request

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--rpc', required=True)
p.add_argument('--output', type=Path, required=True)
p.add_argument('--legacy-ref', required=True, help='pre-upgrade adapter commit')
a = p.parse_args()
header = checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBlockByNumber', 'params': ['latest', False]}, 15))
block = {'blockHash': header['hash'], 'requireCanonical': True}


def lens(code, overrides=None):
    params = [{'data': '0x' + code.hex() + PROXY[2:].rjust(64, '0'), 'gas': hex(3_000_000), 'gasPrice': '0x1'}, block]
    if overrides is not None:
        params.append(overrides)
    return request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call', 'params': params}, 15)


response = lens(CODE)
payload = response.get('error', {}).get('data', '')
assert isinstance(payload, str) and payload.startswith('0x54484738'), response
raw = bytes.fromhex(payload[2:])
word = lambda offset: int.from_bytes(raw[offset:offset + 32], 'big')
stock_offset, shares_offset = word(128), word(160)
count = word(stock_offset)
assert count == word(shares_offset) == 9
assert word(32) == int(header['number'], 16)
overrides = {}
for i in range(count):
    token = '0x' + raw[stock_offset + 32 * (i + 1) + 12:stock_offset + 32 * (i + 2)].hex()
    shares = raw[shares_offset + 32 * (i + 1):shares_offset + 32 * (i + 2)]
    # sharesOf selector -> observed shares. Every other call reverts.
    code = bytes.fromhex('60003560e01c63f5eb42dc146012575f5ffd5b7f') + shares + bytes.fromhex('5f5260205ff3')
    overrides[token] = {'code': '0x' + code.hex()}
isolated = lens(CODE, overrides)
assert isolated.get('error', {}).get('data') == payload, isolated
legacy = bytes.fromhex(subprocess.check_output(['git', 'show', a.legacy_ref + ':pkg/liquidity-source/thog-prop/abi/ThogSnapshotLens.bin'], text=True).strip())
legacy_response = lens(legacy, overrides)
legacy_data = legacy_response.get('error', {}).get('data', '')
assert isinstance(legacy_data, str) and '2b63c300' in legacy_data and not legacy_data.startswith('0x54484738'), legacy_response
capture = {
    'chainId': int(checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_chainId', 'params': []}, 15)), 16),
    'proxy': PROXY, 'blockHash': header['hash'], 'blockNumber': int(header['number'], 16),
    'lensSha256': hashlib.sha256(CODE).hexdigest(), 'response': response,
    'issuerGetterIsolation': {'overrides': overrides, 'response': isolated,
        'legacyRef': a.legacy_ref, 'legacyLensSha256': hashlib.sha256(legacy).hexdigest(), 'legacyResponse': legacy_response},
}
assert capture['chainId'] == 143
assert checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBlockByNumber', 'params': [header['number'], False]}, 15))['hash'] == header['hash']
a.output.write_bytes(gzip.compress((json.dumps(capture, separators=(',', ':')) + '\n').encode(), mtime=0))
print(f"Captured block {capture['blockNumber']}: nine stocks, identical stored-multiplier result when issuer getters revert; old lens rejects.")
