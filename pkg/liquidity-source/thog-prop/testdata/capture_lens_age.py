#!/usr/bin/env python3
"""Capture age-15 regression cases from deployed bytecode on an isolated Monad Anvil fork."""
import gzip
import json
import pathlib
import sys
import urllib.parse
import urllib.request

source, endpoint, output = sys.argv[1:]
assert urllib.parse.urlparse(endpoint).hostname in ('127.0.0.1', 'localhost', '::1'), 'local fork only'
capture = json.loads(gzip.decompress(pathlib.Path(source).read_bytes()))
payload = bytes.fromhex(capture['response']['error']['data'][2:])
word = lambda data, at: int.from_bytes(data[at:at + 32], 'big')
page_at = word(payload, 96)
page = payload[page_at + 32:page_at + 32 + word(payload, page_at)]
base = word(page, 0)
block = word(page, base + 32)
words_at = base + word(page, base + 320)
words = {word(page, words_at + 32 + i * 64): word(page, words_at + 64 + i * 64)
         for i in range(word(page, words_at))}
control = (words[2] & ~(255 << 138)) | (15 << 138)
tokens_at = base + word(page, base + 7 * 32)
tokens = ['0x' + page[tokens_at + 32 + i * 15 * 32 + 12:tokens_at + 64 + i * 15 * 32].hex()
          for i in range(word(page, tokens_at))]
proxy = capture['proxy']

def rpc(request):
    req = urllib.request.Request(endpoint, json.dumps(request).encode(), {'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=60) as response:
        return json.load(response)

version = rpc({'jsonrpc': '2.0', 'id': 1, 'method': 'web3_clientVersion', 'params': []})
assert 'anvil' in version['result'].lower(), version
head = rpc({'jsonrpc': '2.0', 'id': 1, 'method': 'eth_blockNumber', 'params': []})
assert int(head['result'], 16) == block, 'fork the exact captured lens block'
requests, cases = [], []
for input_index, output_index in [(0, 3), (0, 8)]:
    for elapsed_blocks in range(16):
        calldata = '0xa42b1498' + tokens[input_index][2:].rjust(64, '0') + tokens[output_index][2:].rjust(64, '0') + f'{1_000_000:064x}'
        case = {'inputIndex': input_index, 'outputIndex': output_index, 'amount': '1000000',
                'elapsedMs': elapsed_blocks * 300, 'quoteBlock': block + elapsed_blocks + 1}
        cases.append(case)
        requests.append({'jsonrpc': '2.0', 'id': len(cases) - 1, 'method': 'eth_call',
                         'params': [{'to': proxy, 'data': calldata, 'gasPrice': '0x1'}, 'latest',
                                    {proxy: {'stateDiff': {'0x' + f'{2:064x}': '0x' + f'{control:064x}'}}},
                                    {'number': hex(case['quoteBlock'])}]})
for start in range(0, len(requests), 8):
    for reply in rpc(requests[start:start + 8]):
        if 'error' in reply:
            assert reply['error']['code'] == 3, reply
        cases[reply['id']].update({key: value for key, value in reply.items() if key in ('result', 'error')})
fixture = {'proxy': proxy, 'lensPayload': capture['response']['error']['data'], 'blockNumber': block,
           'maxAge': 15, 'controlOverride': hex(control), 'cases': cases,
           'provenance': 'Isolated Anvil --network monad fork at the lens block; only CONTROL.maxAge=15 and evaluation block are overridden. No transactions broadcast.'}
pathlib.Path(output).write_bytes(gzip.compress(json.dumps(fixture, separators=(',', ':')).encode(), mtime=0))
print(f'Saved {len(cases)} EVM cases: {sum("result" in case for case in cases)} successes; {output}')
