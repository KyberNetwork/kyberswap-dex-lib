const {Interface,AbiCoder}=require('ethers');
// Disposable local Anvil fixtures. Caller owns the write-denying upstream.
module.exports=async function({send,account,fee,gateway}){
 const gatewayAddress='0x341649D9A20fAf349aB8F1a1a4449bDa47cFAb35';
 const parent='0x3245af253dc425459c331b2ff8b2fe170d781ae5';
 async function launch(quote,threshold){
  const data=gateway.encodeFunctionData('launch',[[quote,'KYBER NESTED LOCAL','KYBERNEST',['','','','',''],10n**29n,threshold*3n/10n,threshold,true],[0,0,Math.floor(Date.now()/1000)+3600,'0x',0,'0x']]);
  const receipt=await send({from:account,to:gatewayAddress,data,value:'0x'+fee.toString(16),gas:'0x1c9c380'});
  for(const log of receipt.logs){try{const e=gateway.parseLog(log);if(e?.name==='NativeLaunched')return e.args.token.toLowerCase();}catch{}}
  throw Error('Missing nested launch event');
 }
 const settlement=new Interface(['function buyWithRefund(address,uint256,uint256,uint256,bytes,uint256,bytes) payable returns(uint256)']);
 const nestedParent=await launch(parent,10n**24n);
 const route=AbiCoder.defaultAbiCoder().encode(['bytes4','address[]','bytes'],['0x46574c31',[parent],'0x']);
 await send({from:account,to:'0xaD06B86264411e0278DbCebce556c913b44dA004',data:settlement.encodeFunctionData('buyWithRefund',[nestedParent,1,1,Math.floor(Date.now()/1000)+3600,route,1,route]),value:'0x'+(10n**11n).toString(16),gas:'0x1c9c380'});
 return {FLYWHEEL_LOCAL_PARENT:parent,FLYWHEEL_LOCAL_NESTED_PARENT:nestedParent,
  FLYWHEEL_LOCAL_CHILD:await launch(parent,10n**32n),FLYWHEEL_LOCAL_CHILD_REFUND:await launch(parent,10n**24n),
  FLYWHEEL_LOCAL_GRANDCHILD:await launch(nestedParent,10n**32n),FLYWHEEL_LOCAL_GRANDCHILD_REFUND:await launch(nestedParent,10n**24n)};
};
