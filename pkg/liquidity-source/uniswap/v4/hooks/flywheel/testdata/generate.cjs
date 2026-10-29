// All writes are restricted to an ephemeral local Anvil fork. No private keys are loaded.
const fs=require('node:fs'),path=require('node:path'),http=require('node:http'),assert=require('node:assert/strict'),{spawn}=require('node:child_process');
const {ethers}=require('ethers');
const root=__dirname;
const deployment=JSON.parse(fs.readFileSync(path.join(root,'deployment.json'))),a=deployment.addresses,c=deployment.canonical;
const proof=path.join(__dirname,'.fork-proof');fs.mkdirSync(proof,{recursive:true});
const cacheFile=path.join(proof,'public-rpc-cache.json');
const cache=new Map(fs.existsSync(cacheFile)?Object.entries(JSON.parse(fs.readFileSync(cacheFile))):[]);
const allowed=new Set(['eth_chainId','net_version','eth_getBlockByNumber','eth_getBlockByHash','eth_getCode','eth_getStorageAt','eth_getBalance','eth_getTransactionCount','eth_call','eth_getLogs','eth_getTransactionByHash','eth_getTransactionReceipt','eth_gasPrice']);
let reads=0,blocked=0;
async function upstream(item){
 if(!allowed.has(item.method)){blocked++;throw Error('Upstream write or unapproved method blocked: '+item.method);}
 const key=JSON.stringify([item.method,item.params]);
 if(item.params?.includes('latest'))cache.delete(key);
 if(!cache.has(key)){
  if(++reads>2000)throw Error('Public RPC read budget exceeded');
  const res=await fetch('https://rpc.mainnet.chain.robinhood.com',{method:'POST',headers:{'content-type':'application/json'},body:JSON.stringify({...item,jsonrpc:'2.0',id:1}),signal:AbortSignal.timeout(20000)});
  if(!res.ok)throw Error('Public RPC HTTP '+res.status);const data=await res.json();if(data.error)throw Error('Public RPC '+JSON.stringify(data.error));cache.set(key,data.result);
 }
 return{jsonrpc:'2.0',id:item.id,result:cache.get(key)};
}
const json=x=>JSON.stringify(x,(_,v)=>typeof v==='bigint'?v.toString():v,2);
const raw=x=>JSON.stringify(x,(_,v)=>typeof v==='bigint'?'#BIG#'+v:v).replace(/"#BIG#(-?\d+)"/g,'$1');
const erc=['function balanceOf(address) view returns(uint256)','function decimals() view returns(uint8)','function approve(address,uint256) returns(bool)','function deposit() payable'];
const kt='tuple(address currency0,address currency1,uint24 fee,int24 tickSpacing,address hooks)';
const pt='tuple(address intermediateCurrency,uint24 fee,int24 tickSpacing,address hooks,bytes hookData)[]';
const coder=ethers.AbiCoder.defaultAbiCoder(),lower=x=>x.toLowerCase(),E=10n**18n;
async function main(){
 assert.equal(BigInt((await upstream({id:1,method:'eth_chainId',params:[]})).result),4663n);
 // Fixed snapshot makes the test repeatable and allows persistent public-RPC caching.
 const forkBlock=process.env.FORK_BLOCK?Number(process.env.FORK_BLOCK):Number(BigInt((await upstream({id:2,method:'eth_getBlockByNumber',params:['latest',false]})).result.number));
 const block=(await upstream({id:2,method:'eth_getBlockByNumber',params:[ethers.toQuantity(forkBlock),false]})).result;assert(block?.hash);
 const server=http.createServer(async(req,res)=>{let body='';for await(const b of req)body+=b;let data;try{data=JSON.parse(body);const out=Array.isArray(data)?await Promise.all(data.map(upstream)):await upstream(data);res.end(JSON.stringify(out));}catch(e){res.end(JSON.stringify({jsonrpc:'2.0',id:data?.id??null,error:{code:-32000,message:e.message}}));}});
 await new Promise(r=>server.listen(0,'127.0.0.1',r));
 const port=Number(process.env.ANVIL_PORT||18647),child=spawn(process.env.ANVIL || 'anvil',['--fork-url','http://127.0.0.1:'+server.address().port,'--fork-block-number',String(forkBlock),'--port',String(port),'--chain-id','4663','--silent'],{windowsHide:true,stdio:['ignore','pipe','pipe']});
 let anvilOutput='';for(const stream of [child.stdout,child.stderr])stream.on('data',b=>{anvilOutput+=b;});
 const rpc=new ethers.JsonRpcProvider('http://127.0.0.1:'+port,4663,{staticNetwork:true,batchMaxCount:1,cacheTimeout:-1});rpc.pollingInterval=50;
 try{
  for(let i=0;i<100;i++){try{await rpc.send('eth_blockNumber',[]);break;}catch{if(i===99)throw Error(anvilOutput);await new Promise(r=>setTimeout(r,100));}}
  const hashes={};for(const [key,address] of Object.entries({factory:a.factory,hook:a.hook,manager:c.manager,quoter:c.quoter,router:c.router,view:c.view})){hashes[key]=ethers.keccak256(await rpc.getCode(address));assert.equal(hashes[key],deployment.runtimeHashes[key]||deployment.dependencyRuntimeHashes[key]);}
  const signer=await rpc.getSigner(0),actor=await signer.getAddress();
  await rpc.send('anvil_setBalance',[actor,ethers.toQuantity(10000n*E)]);
  await rpc.send('anvil_impersonateAccount',[deployment.owner]);await rpc.send('anvil_setBalance',[deployment.owner,ethers.toQuantity(100n*E)]);const owner=await rpc.getSigner(deployment.owner);
  const abi=name=>JSON.parse(fs.readFileSync(path.join(root,'abi',name+'.json')));
  const factory=new ethers.Contract(a.factory,abi('StandardLaunchFactory'),signer),hook=new ethers.Contract(a.hook,abi('StandardLaunchHook'),rpc),quoter=new ethers.Contract(c.quoter,abi('V4Quoter'),rpc);
  const view=new ethers.Contract(c.view,['function getSlot0(bytes32) view returns(uint160,int24,uint24,uint24)','function getLiquidity(bytes32) view returns(uint128)','function getTickBitmap(bytes32,int16) view returns(uint256)','function getTickInfo(bytes32,int24) view returns(uint128,int128,uint256,uint256)'],rpc);
  const router=new ethers.Contract(c.router,['function execute(bytes,bytes[],uint256) payable'],signer),permit=new ethers.Contract(c.permit2,['function approve(address,address,uint160,uint48)'],signer);
  const weth=new ethers.Contract(c.weth,erc,signer);await(await weth.deposit({value:100n*E})).wait();
  const fixtures=[];
  async function snapshot(token){
   const b=Number(BigInt(await rpc.send('eth_blockNumber',[]))),opts={blockTag:b};
   const k=await hook.poolKey(token,opts),key={currency0:lower(k.currency0),currency1:lower(k.currency1),fee:Number(k.fee),tickSpacing:Number(k.tickSpacing),hooks:lower(k.hooks)},id=ethers.keccak256(coder.encode([kt],[key]));
   const s=await view.getSlot0(id,opts),liquidity=await view.getLiquidity(id,opts),m=await hook.markets(id,opts),principal=await hook.principal(token,opts);
   assert.equal(s[2],0n,'fixture assumes zero protocol fee');assert.equal(s[3],0n);
   const ticks=[];for(let w=-18;w<=17;w++){let bits=await view.getTickBitmap(id,w,opts);for(let bit=0;bits>0n;bit++,bits>>=1n)if(bits&1n){const index=(w*256+bit)*200,t=await view.getTickInfo(id,index,opts);ticks.push({index,liquidityGross:t[0],liquidityNet:t[1]});}}
   assert.equal(ticks.length,2,'fixture reserve calculation requires only the seed position');assert.equal(ticks[0].liquidityGross,m.liquidity);assert.equal(ticks[1].liquidityGross,m.liquidity);
   const tokens=[];for(const address of [key.currency0,key.currency1])tokens.push({address,decimals:Number(await new ethers.Contract(address,erc,rpc).decimals(opts)),swappable:true});
   const reserves=key.currency0===lower(token)?[principal[0],principal[1]]:[principal[1],principal[0]];
   const pool={address:id,type:'uniswap-v4',exchange:'uniswap-v4',blockNumber:b,swapFee:0,tokens,reserves:reserves.map(String),extra:raw({liquidity,sqrtPriceX96:s[0],tick:Number(s[1]),tickSpacing:200,ticks}),staticExtra:JSON.stringify({'0x0':[false,false],fee:0,tS:200,hooks:lower(a.hook),uR:lower(c.router),pm2:lower(c.permit2)})};
   return{pool,key,graduated:m.graduated};
  }
  async function trade(label,token,input,amount,who=signer){
   const snap=await snapshot(token),out=lower(input)===snap.key.currency0?snap.key.currency1:snap.key.currency0,z=lower(input)===snap.key.currency0;
   const [quoted,quoteGas]=await quoter.quoteExactInputSingle.staticCall([snap.key,z,amount,'0x']);assert(quoted>0n);
   const payer=await who.getAddress(),iin=new ethers.Contract(input,erc,who),o=new ethers.Contract(out,erc,rpc);
   await(await iin.approve(c.permit2,ethers.MaxUint256)).wait();await(await permit.connect(who).approve(input,c.router,(1n<<160n)-1n,(1n<<48n)-1n)).wait();
   const hop=[[out,0,200,a.hook,'0x']],swap=coder.encode([`tuple(address currency,${pt} path,uint256[] minHopPriceX36,uint128 amount,uint128 limit)`],[[input,hop,[],amount,quoted]]);
   const actions=coder.encode(['bytes','bytes[]'],['0x070b0e',[swap,coder.encode(['address','uint256','bool'],[input,0,true]),coder.encode(['address','address','uint256'],[out,payer,0])]]);
   const beforeIn=await iin.balanceOf(payer),beforeOut=await o.balanceOf(payer);
   const receipt=await(await router.connect(who).execute('0x10',[actions],ethers.MaxUint256,{gasLimit:2000000})).wait();
   const spent=beforeIn-await iin.balanceOf(payer),received=await o.balanceOf(payer)-beforeOut;assert.equal(received,quoted);assert.equal(spent,amount);
   const logs=receipt.logs.map(l=>{try{return hook.interface.parseLog(l)}catch{return null}}),fee=logs.find(l=>l?.name==='SwapFeeAccrued');assert(fee);assert.equal(fee.args[4],fee.args[3]/25n);assert.equal(received,fee.args[3]-fee.args[4]);
   const post=await snapshot(token),next=(await quoter.quoteExactInputSingle.staticCall([post.key,z,amount,'0x']))[0];
   const trace=await rpc.send('debug_traceTransaction',[receipt.hash,{tracer:'callTracer'}]);let hookGas=0;function walk(x){if(lower(x.to||'')===lower(a.hook)&&x.input?.startsWith(hook.interface.getFunction('afterSwap').selector))hookGas+=Number(BigInt(x.gasUsed));for(const y of x.calls||[])walk(y);}walk(trace);
   fixtures.push({label,pool:snap.pool,tokenIn:lower(input),tokenOut:out,amountIn:String(amount),amountOut:String(quoted),fee:String(fee.args[4]),graduatedBefore:snap.graduated,graduatedAfter:post.graduated,postPool:post.pool,nextAmountOut:String(next),gasUsed:String(receipt.gasUsed),quoterGas:String(quoteGas),afterSwapGas:hookGas,localTransaction:receipt.hash});
   console.log(label+': quote/execution matched; hook gas '+hookGas+'; graduated '+post.graduated);return received;
  }
  const live=[['weth','0x311e8e9eaDC4719784FD8e62A70B7db9B8CDA3f7',c.weth,10n**12n,signer],['usdg','0x5aF1b667311A94959b17Cbf9FD8a65d96cB8cDBB','0x5fc5360D0400a0Fd4f2af552ADD042D716F1d168',1000n,owner],['cashcat','0x737345A546021B8447aF5e3082C10c12A8E2F27a','0x020bfC650A365f8BB26819deAAbF3E21291018b4',10n**14n,owner]];
  for(const[label,token,quote,amount,who]of live){const out=await trade(label+'-buy',token,quote,amount,who);await trade(label+'-sell',token,token,out/2n,who);}
  const receipt=await(await factory.launch(c.weth,'Local integration test','LOCAL',['','','','',''],100000000000n*E,E,3n*E,true,{value:500000000000000n,gasLimit:9000000})).wait();
  const token=receipt.logs.map(l=>{try{return factory.interface.parseLog(l)}catch{return null}}).find(l=>l?.name==='TokenLaunched').args.token;
  await trade('first-buy',token,c.weth,10n**12n);
  const large=await trade('single-buy-graduation',token,c.weth,4n*E);assert(fixtures.at(-1).graduatedAfter);assert(!fixtures.at(-1).graduatedBefore);
  await trade('graduated-sell',token,token,large/10n);
  await trade('graduated-buy',token,c.weth,E/100n);
  const data={chainId:4663,forkBlock,forkBlockHash:block.hash,upstream:'https://rpc.mainnet.chain.robinhood.com',mainnetTransactions:0,runtimeHashes:hashes,factory:a.factory,hook:a.hook,manager:c.manager,quoter:c.quoter,router:c.router,reads,fixtures};
  fs.writeFileSync(path.join(__dirname,'fork-fixtures.json'),json(data));console.log('PASS '+fixtures.length+' local executions; public upstream reads '+reads+'; paid Alchemy reads 0.');
 }finally{rpc.destroy();child.kill();server.closeAllConnections();await new Promise(r=>server.close(r));fs.writeFileSync(cacheFile,JSON.stringify(Object.fromEntries(cache)));fs.writeFileSync(path.join(proof,'anvil.log'),anvilOutput);}
}
main().catch(e=>{console.error(e.stack||e.message);process.exitCode=1});
