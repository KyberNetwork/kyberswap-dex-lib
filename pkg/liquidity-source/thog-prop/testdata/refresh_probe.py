#!/usr/bin/env python3
"""Read-only ThogAMM provider comparison. No transactions are signed or broadcast."""
import argparse
import concurrent.futures
import gzip
import hashlib
import json
import pathlib
import statistics
import time
import urllib.parse
import urllib.request

PROXY = '0x80c74517bcc2d67ffe02d3ed886796272f647210'
PAGE = '0x6a7373d6' + '0' * 64 + f'{28:064x}'
ISSUERS = {'0x' + x for x in (
    '90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48', 'a753a7395cae905cd615da0b82a53e0560f250af',
    '8d512a03600981b8e86556fd5c0fbdd726fe1821', '299c95eb52bfd5bc98d924bce17197b64864589b',
    'c845b2894dbddd03858fd2d643b4ef725fe0849d', '8ad3c73f833d3f9a523ab01476625f269aeb7cf0',
    '68fa48b1c2fe52b3d776e1953e0e782b5044ce28', 'f6a873bae4ba1b304e45df52a4b7d176e1c6a8c4',
    '58100046a4afcd4ee4fadbd4244f3f895a341c56')}
ROOT = pathlib.Path(__file__).resolve().parent.parent
CODE = bytes.fromhex((ROOT / 'abi/ThogSnapshotLens.bin').read_text().strip())


def request(url, payload, timeout):
    req = urllib.request.Request(url, json.dumps(payload).encode(), {'Content-Type': 'application/json'})
    with urllib.request.urlopen(req, timeout=timeout) as response:
        return json.load(response)


def checked(reply):
    if 'error' in reply:
        raise RuntimeError(json.dumps(reply['error']))
    return reply['result']


def call(to, data, block):
    return {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call',
            'params': [{'to': to, 'data': data, 'gasPrice': '0x1'}, block]}


def page_context(raw):
    data = bytes.fromhex(raw[2:])
    word = lambda at: int.from_bytes(data[at:at + 32], 'big')
    base = word(0)
    tokens = base + word(base + 7 * 32)
    addresses = ['0x' + data[tokens + 32 + i * 15 * 32 + 12:tokens + 64 + i * 15 * 32].hex()
                 for i in range(word(tokens))]
    return word(base + 32), word(base + 64), '0x' + data[base + 96:base + 128].hex(), addresses


def run_read(url, design, timeout):
    if design == 'lens':
        init = '0x' + CODE.hex() + PROXY[2:].rjust(64, '0')
        reply = request(url, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_call',
                             'params': [{'data': init, 'gas': hex(3_000_000), 'gasPrice': '0x1'}, 'latest']}, timeout)
        payload = reply.get('error', {}).get('data', '')
        if not isinstance(payload, str) or not payload.startswith('0x54484738'):
            raise RuntimeError(json.dumps(reply))
        return reply
    header = None
    block = 'latest'
    if design in ('hash', 'number'):
        header = checked(request(url, {'jsonrpc': '2.0', 'id': 1, 'method': 'eth_getBlockByNumber',
                                       'params': ['latest', False]}, timeout))
        block = {'blockHash': header['hash'], 'requireCanonical': True} if design == 'hash' else header['number']
    raw = checked(request(url, call(PROXY, PAGE, block), timeout))
    number, fee, parent, tokens = page_context(raw)
    if header:
        assert (number, fee, parent) == (int(header['number'], 16), int(header['baseFeePerGas'], 16), header['parentHash'])
    if design != 'hash':
        block = hex(number)
    batch = []
    for token in tokens:
        if token in ISSUERS:
            batch.extend([call(token, '0xf5eb42dc' + PROXY[2:].rjust(64, '0'), block), call(token, '0x2b63c300', block)])
    # The deployed page-first reader confirms the same page in the issuer batch.
    if design == 'page':
        batch.append(call(PROXY, PAGE, block))
    for i, item in enumerate(batch):
        item['id'] = i
    replies = request(url, batch, timeout)
    results = {item['id']: checked(item) for item in replies}
    assert len(results) == len(batch)
    if design == 'page':
        assert results[len(batch) - 1] == raw, 'page changed at pinned number'
    return None


def probe(url, args):
    rows = []
    for design in ('hash', 'number', 'page', 'lens'):
        elapsed, errors = [], []
        for _ in range(args.count):
            started = time.perf_counter()
            captured_at = int(time.time() * 1000)
            try:
                reply = run_read(url, design, args.timeout)
                elapsed.append(round((time.perf_counter() - started) * 1000, 1))
                if design == 'lens' and args.capture:
                    data = {'proxy': PROXY, 'capturedAtMs': captured_at,
                            'lensSha256': hashlib.sha256(CODE).hexdigest(),
                            'endpoint': urllib.parse.urlparse(url).hostname, 'response': reply}
                    pathlib.Path(args.capture).write_bytes(gzip.compress(json.dumps(data, separators=(',', ':')).encode(), mtime=0))
            except Exception as error:
                errors.append(str(error)[:350])
            time.sleep(args.pause)
        row = {'endpoint': url, 'design': design, 'success': len(elapsed), 'attempts': args.count,
               'medianMs': statistics.median(elapsed) if elapsed else None, 'errors': errors}
        rows.append(row)
        print(json.dumps(row), flush=True)
    return rows


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--rpc', action='append', required=True)
    parser.add_argument('--count', type=int, default=6)
    parser.add_argument('--timeout', type=float, default=2)
    parser.add_argument('--pause', type=float, default=0.5)
    parser.add_argument('--capture', help='Write one successful lens capture as .json.gz (use one RPC).')
    parser.add_argument('--output', required=True)
    args = parser.parse_args()
    if args.capture and len(args.rpc) != 1:
        parser.error('--capture requires exactly one RPC')
    with concurrent.futures.ThreadPoolExecutor(max_workers=len(args.rpc)) as executor:
        rows = list(executor.map(lambda url: probe(url, args), args.rpc))
    pathlib.Path(args.output).write_text(json.dumps({'capturedAt': int(time.time()), 'rows': sum(rows, [])}, indent=2) + '\n')


if __name__ == '__main__':
    main()
