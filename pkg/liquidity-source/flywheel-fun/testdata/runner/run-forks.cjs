'use strict';
// Forge/Anvil execute locally. The upstream proxy denies all writes/signatures.
const {spawn} = require('node:child_process');
const net = require('node:net');
const {Interface} = require('ethers');
const {fs,path,root,dex,adapters,bins,block,run,save,startReadBridge} = require('./common.cjs');
async function main() {
  const mode = process.argv[2];
  if (!['adapter','quotes'].includes(mode)) throw Error('Usage: node run-forks.cjs adapter|quotes');
  let bridge, anvil;
  try {
    // Build before spending RPC requests. No network/signing config is loaded.
    const build = await run(bins.forge, ['build','--evm-version','cancun','--use','0.8.30'], adapters);
    if (build.code) throw Error('Adapter build failed');
    bridge = await startReadBridge(mode === 'adapter' ? 900 : 2500);
    let result;
    if (mode === 'adapter') {
      result = await run(bins.forge, ['test','--match-path','test/adapters/flywheel-fun/*.t.sol','--evm-version','cancun','--use','0.8.30','--fuzz-runs','128'], adapters, {FLYWHEEL_READONLY_FORK_URL:bridge.url});
    } else {
      const probe = net.createServer(); await new Promise(r => probe.listen(0,'127.0.0.1',r));
      const port = probe.address().port; await new Promise(r => probe.close(r));
      anvil = spawn(bins.anvil, ['--host','127.0.0.1','--port',String(port),'--fork-url',bridge.url,'--fork-block-number',String(block),'--chain-id','4663','--silent','--no-storage-caching'], {windowsHide:true,stdio:['ignore','ignore','pipe']});
      let startupFailed = false; anvil.on('error', () => {startupFailed=true;}); anvil.stderr.on('data', () => {});
      const local = 'http://127.0.0.1:'+port;
      async function rpc(method, params=[]) {
        const r = await fetch(local,{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({jsonrpc:'2.0',id:1,method,params}),signal:AbortSignal.timeout(30000)});
        const d = await r.json(); if (d.error) throw Error('Local Anvil call failed: '+method); return d.result;
      }
      let ready = false;
      for (let n=0;n<100;n++) {if(startupFailed)break;try {await rpc('eth_chainId');ready=true;break;}catch{await new Promise(r=>setTimeout(r,200));}}
      if(!ready)throw Error('Local Anvil did not start');
      const accounts = await rpc('eth_accounts');
      async function send(tx) {
        const hash=await rpc('eth_sendTransaction',[tx]);
        for(let n=0;n<100;n++){const r=await rpc('eth_getTransactionReceipt',[hash]);if(r){if(r.status!=='0x1')throw Error('Local test transaction reverted');return r;}await new Promise(r=>setTimeout(r,100));}
        throw Error('Local receipt timeout');
      }
      const artifact=JSON.parse(fs.readFileSync(path.join(adapters,'out/FlywheelNativeAdapter.sol/FlywheelNativeAdapter.json'),'utf8'));
      const deployed=await send({from:accounts[0],data:artifact.bytecode.object,gas:'0x989680'});
      const gateway=new Interface(JSON.parse(fs.readFileSync(path.join(root,'abi/NativeLaunchGateway.json'),'utf8')));
      const factory=new Interface(['function launchFeeWei() view returns(uint256)']);
      const fee=BigInt(await rpc('eth_call',[{to:'0xe7743b4039DBCd05C5242939aA8db274c65fCBfA',data:factory.encodeFunctionData('launchFeeWei')},'latest']));
      const launches=[];
      for(const [quote,threshold] of [['0x0Bd7D308f8E1639FAb988df18A8011f41EAcAD73',100000000000n],['0x73c2dE14C7FA0a57cc2d9722b959eA70B881fFe4',100000000000000000n],['0x39dBED3a2bd333467115dE45665cC57F813C4571',10n**24n]]){
        const data=gateway.encodeFunctionData('launch',[[quote,'KYBER LOCAL QUOTE PARITY','KYBERLOCAL',['','','','',''],10n**29n,threshold*3n/10n,threshold,true],[0,0,Math.floor(Date.now()/1000)+3600,'0x',0,'0x']]);
        const receipt=await send({from:accounts[0],to:'0x341649D9A20fAf349aB8F1a1a4449bDa47cFAb35',data,value:'0x'+fee.toString(16),gas:'0x1c9c380'});
        for(const log of receipt.logs){try{const e=gateway.parseLog(log);if(e?.name==='NativeLaunched')launches.push(e.args.token.toLowerCase());}catch{}}
      }
      if(launches.length!==3)throw Error('Missing local test launches');
      const nested=await require('./nested-fixtures.cjs')({send,account:accounts[0],fee,gateway});
      result=await run(bins.go,['test','./pkg/liquidity-source/flywheel-fun','-run','TestLocalFork','-count=1','-v','-timeout','8m'],dex,{
        ...nested,FLYWHEEL_CAPTURE_FIXTURES:'0',FLYWHEEL_LOCAL_TEST_RPC:local,FLYWHEEL_LOCAL_ADAPTER:deployed.contractAddress,
        FLYWHEEL_LOCAL_WETH_CURVE:launches[0],FLYWHEEL_LOCAL_BOOMER_CURVE:launches[1],FLYWHEEL_LOCAL_PONS_CURVE:launches[2]
      });
    }
    save(mode+'-tests.txt',result.output);
    save(mode+'-run.json',{passed:result.code===0,forkBlock:block,...bridge.stats(),at:new Date().toISOString()});
    console.log(JSON.stringify({mode,passed:result.code===0,...bridge.stats()}));
    process.exitCode=result.code;
  } finally {if(anvil)anvil.kill();if(bridge)await bridge.close();}
}
main().catch(e=>{console.error(e.message);process.exitCode=1;});
