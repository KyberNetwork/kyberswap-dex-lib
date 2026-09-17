// Reproducible, disposable Anvil fork check; no production key is used.
// Requires Node 24 and viem 2.56.1. See ../README.md.
import assert from 'node:assert/strict';
import { writeFile } from 'node:fs/promises';
const { createPublicClient, http, parseAbi, encodeFunctionData, keccak256, stringToHex, parseEther, maxUint256 } = await import(process.env.VIEM_MODULE || 'viem');
const url = process.env.PREMIUM_FORK_RPC_URL || 'http://127.0.0.1:18547';
assert(['127.0.0.1', 'localhost', '[::1]'].includes(new URL(url).hostname), 'requires localhost Anvil');
const client = createPublicClient({ transport: http(url), cacheTime: 0, pollingInterval: 50 });
const rpc = (method, params = []) => client.request({ method, params });
assert((await rpc('web3_clientVersion')).toLowerCase().includes('anvil'), 'requires Anvil');
assert.equal(await client.getChainId(), 4663);
const forkHead = await client.getBlock();
assert(forkHead.number >= 65000000n, 'fork must include the current PRM-pair contracts');
console.log('Fork block:', String(forkHead.number));
// Default Anvil keys already have EIP-7702 code on this chain; use a clean local account.
const trader = '0x' + keccak256(stringToHex('premium.kyber.local-fork-2026-09-17')).slice(-40);
assert(!await client.getCode({ address: trader }), 'test recipient must have no delegated code');
const factory = '0x34fb85af7588db97fd6db6508aa387d0f088564c';
const router = '0x08a59435c8359a45f4f5dc8d91df893cc33daf29';
const weth = '0x0bd7d308f8e1639fab988df18a8011f41eacad73';
const prm = '0xf24f8f6b08fe87cf062e833a732ad7f636064bc8';
const phodo = '0x64e8bee350c0ba7c7601f7eca7f61ab965148b51';
const holder = '0x07ec901873cff7ad597b9b17b99f17ef04a8af75';
const factoryABI = parseAbi([
  'function createMeme(bytes32,string,string,string) returns(address,address)',
  'function memeCount() view returns(uint256)', 'function memeTokens(uint256) view returns(address)',
  'function getMeme(address) view returns((bytes32 subjectId,address token,address curve,address creator,uint256 graduationDesk,uint64 createdAt))',
]);
const erc20 = parseAbi(['function approve(address,uint256) returns(bool)', 'function transfer(address,uint256) returns(bool)', 'function balanceOf(address) view returns(uint256)']);
const routerABI = parseAbi(['function buyExactIn(address,uint256,uint256) payable returns(uint256)', 'function sellExactIn(address,uint256,uint256) returns(uint256)', 'function buyMemeWithEth(address,uint256,address,uint256) payable returns(uint256,uint256)']);
const curveABI = parseAbi(['function getReserves() view returns(uint256,uint256)', 'function deskRaised() view returns(uint256)', 'function memeSold() view returns(uint256)', 'function phase() view returns(uint8)', 'function deskToken() view returns(address)', 'function isNativeQuote() view returns(bool)', 'function quoteBuy(uint256) view returns(uint256,uint256,uint256)', 'function quoteSell(uint256) view returns(uint256,uint256)']);
const read = (address, abi, functionName, args = [], blockNumber) => client.readContract({ address, abi, functionName, args, blockNumber });
const balance = (token, account = trader) => read(token, erc20, 'balanceOf', [account]);
async function send(address, abi, functionName, args = [], value = 0n, from = trader) {
  const block = await client.getBlock();
  await rpc('evm_setNextBlockTimestamp', [Number(block.timestamp + 1n)]);
  const hash = await rpc('eth_sendTransaction', [{ from, to: address, data: encodeFunctionData({ abi, functionName, args }), value: '0x' + value.toString(16), gas: '0x989680', gasPrice: '0x174876e800', nonce: '0x' + (await client.getTransactionCount({ address: from, blockTag: 'latest' })).toString(16) }]);
  const receipt = await client.waitForTransactionReceipt({ hash, timeout: 60000 });
  assert.equal(receipt.status, 'success', `${functionName} reverted: ${hash}`);
  return receipt;
}
const snapshot = await rpc('evm_snapshot');
try {
  await rpc('anvil_impersonateAccount', [trader]);
  await rpc('anvil_setBalance', [trader, '0x' + parseEther('10000').toString(16)]);
  // Create a PRM-paired market at the real fork price; this is local only.
  const count = await read(factory, factoryABI, 'memeCount');
  await send(factory, factoryABI, 'createMeme', [keccak256(stringToHex('premium.pair.PRM')), 'Local Kyber pair verification', 'LOCAL', '']);
  const child = await read(factory, factoryABI, 'memeTokens', [count]);
  // Acquire PRM through the deployed router and fund pHODO via a local impersonation.
  await send(router, routerABI, 'buyMemeWithEth', [prm, 1n, trader, forkHead.timestamp + 1000n], parseEther('10'));
  await rpc('anvil_impersonateAccount', [holder]);
  await rpc('anvil_setBalance', [holder, '0x56bc75e2d63100000']);
  await send(phodo, erc20, 'transfer', [trader, await balance(phodo, holder)], 0n, holder);
  await rpc('anvil_stopImpersonatingAccount', [holder]);
  const markets = [
    { name: 'pHODO', token: '0x9ec302fed8c4bb95e210a6a8f37b29a15e6d2995', pair: phodo, native: false },
    { name: 'ETH', token: '0x54c1fa485f182a17b3ba9190e7ed481b6b60109e', pair: weth, native: true },
    { name: 'PRM', token: child.toLowerCase(), pair: prm, native: false },
  ];
  const fixtures = [], settlements = [];
  for (const market of markets) {
    const m = await read(factory, factoryABI, 'getMeme', [market.token]);
    const curve = m.curve.toLowerCase(), target = m.graduationDesk;
    assert.equal((await read(curve, curveABI, 'deskToken')).toLowerCase(), market.pair);
    assert.equal(await read(curve, curveABI, 'isNativeQuote'), market.native);
    if (!market.native) await send(market.pair, erc20, 'approve', [router, maxUint256]);
    await send(market.token, erc20, 'approve', [router, maxUint256]);
    let bought = 0n;
    for (const action of ['buy', 'sell', 'final-buy']) {
      const block = await client.getBlock();
      const [reserves, sold, raised, phase] = await Promise.all([
        read(curve, curveABI, 'getReserves', [], block.number), read(curve, curveABI, 'memeSold', [], block.number),
        read(curve, curveABI, 'deskRaised', [], block.number), read(curve, curveABI, 'phase', [], block.number),
      ]);
      const input = action === 'buy' ? target / 1000n : action === 'sell' ? bought / 2n : ((target - raised) * 10000n + 9899n) / 9900n + target / 100n;
      const buy = action !== 'sell';
      const quote = await read(curve, curveABI, buy ? 'quoteBuy' : 'quoteSell', [input], block.number);
      const [output, used, fee] = buy ? quote : [quote[0], input, quote[1]];
      assert(output > 0n);
      const beforePair = market.native ? await client.getBalance({ address: trader }) : await balance(market.pair);
      const beforeMeme = await balance(market.token);
      const tx = await send(router, routerABI, buy ? 'buyExactIn' : 'sellExactIn', [market.token, input, output], buy && market.native ? input : 0n);
      const afterPair = market.native ? await client.getBalance({ address: trader }) : await balance(market.pair);
      const afterMeme = await balance(market.token);
      const gasCost = market.native ? tx.gasUsed * tx.effectiveGasPrice : 0n;
      assert.equal(buy ? beforePair - afterPair - gasCost : afterPair - beforePair + gasCost, buy ? used : output, `${market.name} ${action} pair/refund`);
      assert.equal(buy ? afterMeme - beforeMeme : beforeMeme - afterMeme, buy ? output : used, `${market.name} ${action} meme`);
      if (action === 'buy') bought = output;
      if (action === 'final-buy') {
        assert(input > used);
        assert.equal(await read(curve, curveABI, 'phase'), 1);
      }
      const p = {
        address: curve, exchange: 'prm-fun', type: 'prm-fun', timestamp: Number(block.timestamp),
        reserves: reserves.map(String), tokens: [{ address: market.pair, swappable: true }, { address: market.token, swappable: true }],
        extra: JSON.stringify({ ph: phase, vM: String(reserves[1]), vD: String(reserves[0]), mS: String(sold), dR: String(raised) }),
        staticExtra: JSON.stringify({ rA: router, cA: curve, mT: market.token, gD: String(target), nQ: market.native }), blockNumber: Number(block.number),
      };
      const after = await Promise.all([read(curve, curveABI, 'getReserves'), read(curve, curveABI, 'memeSold'), read(curve, curveABI, 'deskRaised'), read(curve, curveABI, 'phase')]);
      fixtures.push({ pair: market.name + '/' + action, pool: p, quotes: [{ buy, input: String(input), output: String(output), used: String(used), fee: String(fee) }], after: { ph: after[3], vM: String(after[0][1]), vD: String(after[0][0]), mS: String(after[1]), dR: String(after[2]) } });
      const evidence = { pair: market.name, action, input: String(input), used: String(used), output: String(output), refund: String(buy ? input - used : 0n), gasUsed: String(tx.gasUsed), localTx: tx.transactionHash };
      settlements.push(evidence);
      console.log(JSON.stringify(evidence));
    }
  }
  await writeFile(new URL('./fork-quotes.json', import.meta.url), JSON.stringify(fixtures, null, 2) + '\n');
  await writeFile(new URL('./fork-settlement.json', import.meta.url), JSON.stringify({ environment: 'local Anvil fork; transactions are not on mainnet', chainId: 4663, forkBlock: Number(forkHead.number), forkHash: forkHead.hash, settlements }, null, 2) + '\n');
} finally {
  await rpc('anvil_stopImpersonatingAccount', [trader]);
  await rpc('evm_revert', [snapshot]);
}
