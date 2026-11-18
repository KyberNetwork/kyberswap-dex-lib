#!/usr/bin/env python3
"""Capture the snapshot lens and show that it asks a share-based token for sharesOf alone.

All calls are read-only. The isolation case overrides only token code inside
eth_call: sharesOf returns the observed shares, and every other selector reverts.
"""
import argparse
import gzip
import hashlib
import json
from pathlib import Path
from refresh_probe import CODE, PROXY, checked, request

p = argparse.ArgumentParser(description=__doc__)
p.add_argument('--rpc', required=True)
p.add_argument('--output', type=Path, required=True)
a = p.parse_args()
header = checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBlockByNumber', 'params': ['latest', False]}, 15))
block = {'blockHash': header['hash'], 'requireCanonical': True}


def lens(overrides=None):
    params = [{'data': '0x' + CODE.hex() + PROXY[2:].rjust(64, '0'), 'gas': hex(3_000_000), 'gasPrice': '0x1'}, block]
    if overrides is not None:
        params.append(overrides)
    return request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call', 'params': params}, 15)


response = lens()
payload = response.get('error', {}).get('data', '')
assert isinstance(payload, str) and payload.startswith('0x54484738'), response
raw = bytes.fromhex(payload[2:])
word = lambda offset: int.from_bytes(raw[offset:offset + 32], 'big')
token_offset, shares_offset = word(128), word(160)
count = word(token_offset)
assert count == word(shares_offset) and count > 0
assert word(32) == int(header['number'], 16)
overrides = {}
for i in range(count):
    token = '0x' + raw[token_offset + 32 * (i + 1) + 12:token_offset + 32 * (i + 2)].hex()
    shares = raw[shares_offset + 32 * (i + 1):shares_offset + 32 * (i + 2)]
    # sharesOf selector -> observed shares. Every other call reverts.
    code = bytes.fromhex('60003560e01c63f5eb42dc146012575f5ffd5b7f') + shares + bytes.fromhex('5f5260205ff3')
    overrides[token] = {'code': '0x' + code.hex()}
# The endpoint must honour the override: another issuer getter answers normally and reverts under it.
token = next(iter(overrides))
getter = [{'to': token, 'data': '0x2b63c300'}, block]  # getCurrentMultiplier()
assert 'result' in request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call', 'params': getter}, 15)
assert 'error' in request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call', 'params': getter + [overrides]}, 15)
isolated = lens(overrides)
assert isolated.get('error', {}).get('data') == payload, isolated
capture = {
    'chainId': int(checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_chainId', 'params': []}, 15)), 16),
    'proxy': PROXY, 'blockHash': header['hash'], 'blockNumber': int(header['number'], 16),
    'lensSha256': hashlib.sha256(CODE).hexdigest(), 'response': response,
    'issuerGetterIsolation': {'overrides': overrides, 'response': isolated},
}
assert capture['chainId'] == 143
assert checked(request(a.rpc, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBlockByNumber', 'params': [header['number'], False]}, 15))['hash'] == header['hash']
a.output.write_bytes(gzip.compress((json.dumps(capture, separators=(',', ':')) + '\n').encode(), mtime=0))
print(f"Captured block {capture['blockNumber']}: {count} share-based tokens, identical snapshot when they answer sharesOf alone.")
