#!/usr/bin/env python3
"""Capture block-number-only quote overrides from a local Monad fork."""
import collections,gzip,json,pathlib,sys,urllib.parse,urllib.request
source=pathlib.Path(sys.argv[1])
rpc=sys.argv[2]
assert urllib.parse.urlparse(rpc).hostname in ('127.0.0.1','localhost','::1'), 'use a local Anvil fork'
out=pathlib.Path(sys.argv[3])
f=json.loads(gzip.decompress(source.read_bytes()));s=f['snapshots'][0];block=s['blockNumber'];proxy=f['proxy']
raw=bytes.fromhex(s['snapshot'][2:]);word=lambda at:int.from_bytes(raw[at:at+32],'big')
base=word(0);wstart=base+word(base+10*32)
words={word(wstart+32+i*64):word(wstart+64+i*64) for i in range(word(wstart))}
posted=(words[2]>>36)&0xffffffff;maxage=(words[2]>>138)&255
assert (words[2]>>146)&65535==50
groups=collections.defaultdict(list)
for c in s['cases']:
 if c['method']=='quote':groups[(c['inputIndex'],c['outputIndex'])].append(c)
cases=[]
for pair,rows in groups.items():
 for c in [rows[3],rows[5]]:
  cases.append(dict(c,quoteBlock=block+1))
# Inspect actual on-chain age cutoff at representative ordinary and xStock endpoints.
for pair in [(0,1),(0,8),(8,0),(8,9)]:
 for quoteBlock in [posted+maxage,posted+maxage+1]:
  cases.append(dict(groups[pair][3],quoteBlock=quoteBlock))
for c in cases:
 for key in ['success','error','result']:c.pop(key,None)
for offset in range(0,len(cases),16):
 group=cases[offset:offset+16]
 calls=[{'jsonrpc':'2.0','id':i,'method':'eth_call','params':[{'to':proxy,'data':c['calldata'],'gasPrice':'0x1'},'latest',{},{'number':hex(c['quoteBlock'])}]} for i,c in enumerate(group)]
 request=urllib.request.Request(rpc,json.dumps(calls).encode(),{'Content-Type':'application/json'})
 results=json.load(urllib.request.urlopen(request,timeout=60));byid={r['id']:r for r in results}
 for i,c in enumerate(group):
  r=byid[i]
  if 'error' in r:assert r['error']['code']==3,r
  c['success']='result' in r
  c.update({k:v for k,v in r.items() if k in ['result','error']})
 print(offset+len(group),len(cases),flush=True)
s['cases']=cases
f.update(modelVersion=3,quotePolicy='snapshot block + 1; unchanged inventory, valuations and issuer state',capture='Anvil 1.8.6 --network monad; eth_call block-number override; no transactions broadcast',sourceBlockHash=s['blockHash'])
out.write_bytes(gzip.compress(json.dumps(f,separators=(',',':')).encode(),mtime=0))
print('saved',out,'cases',len(cases),'successes',sum(c['success'] for c in cases),flush=True)
