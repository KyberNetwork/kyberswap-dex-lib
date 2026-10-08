'use strict';
const fs = require('node:fs');
const path = require('node:path');
const http = require('node:http');
const { spawn } = require('node:child_process');
const root = __dirname;
const dex = path.resolve(process.env.FLYWHEEL_DEX_LIB_DIR || path.resolve(root, '../../../../..'));
const bins = { go: process.env.GO_BIN || 'go', anvil: process.env.ANVIL_BIN || 'anvil' };
// Archive RPCs can pin the fork with FLYWHEEL_FORK_BLOCK (e.g. 82183340); otherwise fork the head.
const block = process.env.FLYWHEEL_FORK_BLOCK || '';
function save(name, data) {
  const dir = path.resolve(process.env.FLYWHEEL_RESULTS_DIR || path.join(root, '.results')); fs.mkdirSync(dir, {recursive:true});
  fs.writeFileSync(path.join(dir, name), typeof data === 'string' ? data : JSON.stringify(data, null, 2) + '\n');
}
async function run(executable, args, cwd, env = {}) {
  return new Promise((resolve, reject) => {
    const child = spawn(executable, args, {cwd, windowsHide:true, env:{...process.env, ...env}, stdio:['ignore','pipe','pipe']});
    let output = '';
    for (const s of [child.stdout, child.stderr]) s.on('data', b => { output += b; process.stdout.write(b); });
    child.on('error', () => reject(Error('Unable to start ' + path.basename(executable) + '; check tool paths.')));
    child.on('exit', code => resolve({code:code ?? 1, output}));
  });
}
const allowed = new Set(['eth_chainId','net_version','eth_blockNumber','eth_getBlockByNumber','eth_getBlockByHash','eth_getCode','eth_getStorageAt','eth_getBalance','eth_getTransactionCount','eth_call','eth_getLogs','eth_getTransactionByHash','eth_getTransactionReceipt','eth_gasPrice']);
async function startReadBridge(limit = 2500) {
  const upstream = process.env.FLYWHEEL_RPC_URL;
  if (!upstream) throw Error('Set FLYWHEEL_RPC_URL to an archive-capable Robinhood RPC. No signing key is needed.');
  const url = new URL(upstream);
  if (!['http:','https:'].includes(url.protocol)) throw Error('RPC must be HTTP(S).');
  let reads = 0; const denied = {}; const cache = new Map(); const pending = new Map();
  async function rpc(q) {
    if (!allowed.has(q.method)) {
      denied[q.method] = (denied[q.method] || 0) + 1;
      return {jsonrpc:'2.0',id:q.id,error:{code:-32601,message:'Read-only upstream'}};
    }
    const key = JSON.stringify([q.method,q.params]);
    if (!cache.has(key)) {
      if (!pending.has(key)) pending.set(key, (async () => {
        if (++reads > limit) throw Error('Upstream read budget exceeded');
        let response;
        try { response = await fetch(upstream, {method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({...q,id:1}),signal:AbortSignal.timeout(20000)}); }
        catch { throw Error('Upstream read failed or timed out'); }
        if (!response.ok) throw Error('Upstream HTTP read failed');
        const d = await response.json();
        if (d.error) return {error:{code:d.error.code,message:'Upstream RPC read error'}};
        cache.set(key, d.result); return {result:d.result};
      })());
      try { const d = await pending.get(key); if (d.error) return {jsonrpc:'2.0',id:q.id,error:d.error}; }
      finally { pending.delete(key); }
    }
    return {jsonrpc:'2.0',id:q.id,result:cache.get(key)};
  }
  const chain = await rpc({jsonrpc:'2.0',id:1,method:'eth_chainId',params:[]});
  if (!chain.result || BigInt(chain.result) !== 4663n) throw Error('RPC is not Robinhood chain 4663');
  const server = http.createServer(async (req,res) => {
    try {
      let body = '';
      for await (const chunk of req) {body += chunk;if (body.length > 1000000) throw Error('Request too large');}
      const q = JSON.parse(body);
      const result = Array.isArray(q) ? await Promise.all(q.map(rpc)) : await rpc(q);
      res.setHeader('content-type','application/json');res.end(JSON.stringify(result));
    } catch {res.end(JSON.stringify({jsonrpc:'2.0',id:null,error:{code:-32000,message:'Read bridge failed; check RPC availability/budget'}}));}
  });
  await new Promise(r => server.listen(0,'127.0.0.1',r));
  return {url:'http://127.0.0.1:'+server.address().port, rpc,
    stats:() => ({reads,readLimit:limit,deniedMethods:denied,mainnetTransactions:0}),
    close:async () => {server.closeAllConnections();await new Promise(r => server.close(r));}};
}
module.exports = {fs,path,root,dex,bins,block,run,save,startReadBridge};
