# Live-parity fork for the LOTFLOW v1 (lotflow) simulator. LOCAL anvil only (127.0.0.1:8591).
# Deploys the V17 + hook #2 stack on a local fork (needs the Lots contract sources; not part of CI).
# NavJitHookV17 + the MQ stack (hook #2, MqRegistry/MqFactory/NavGuardMq, LOT-two, LOT-Mag 5 and an MQ basket).
# Kyber build / splice / replay parts are not copied. The contracts are a `git archive kd/b contracts` copy outside
# any worktree (env KD_CONTRACTS), so nothing here writes outside its own directory.
#
#   python3 parity.py up <outdir>     start anvil (fork of the PUBLIC 4663 RPC at head-5), deploy, write <outdir>/deploy.json
#   python3 parity.py down <outdir>   stop the anvil this tool started (PID from <outdir>/anvil.pid, checked against :8591)
import json, os, re, subprocess, sys, time, urllib.request

ANVIL = "http://127.0.0.1:8591"
PUBLIC = "https://rpc.mainnet.chain.robinhood.com"
HERE = os.path.dirname(os.path.abspath(__file__))
CONTRACTS = os.environ["KD_CONTRACTS"]
CAST = os.path.expanduser("~/.foundry/bin/cast")
ANVILBIN = os.path.expanduser("~/.foundry/bin/anvil")
FORGE = os.path.expanduser("~/.foundry/bin/forge")
assert ANVIL.startswith("http://127.0.0.1:"), "local anvil only"

KY = "https://aggregator-api.kyberswap.com/robinhood/api/v1"
KH = {"x-client-id": "lots-keeper", "Content-Type": "application/json", "User-Agent": "curl/8.7.1"}
SIG = "swap((address,address,bytes,(address,address,address[],uint256[],address[],uint256[],address,uint256,uint256,uint256,bytes),bytes))"

USDG = "0x5fc5360d0400a0fd4f2af552add042d716f1d168"
USDG_SLOT = "1"  # kx.py SLOT map (balance mapping base slot)
ELI = "0x98c62a9bda22abd049145e1f2dcdcb766dcb2cf4"  # LOT-eli, live V15 venue routed by Kyber as uniswap-v4-navjit
ELI_POOL = "0x33d73de55ffd2a9c1f89e989fdfc19634a3da758398303b1ff89acb6cc3145ee"
V15HOOK = "0x99a670d2103e1e4ebd53a483f725dd651a08eae0"
NAVGUARD15 = "0x033d7000e3ae32fd0a9b1fbaac7c42e26e1e9ea2"
STOCK_REGISTRY = "0x68838b9decf64a67f961733c710d5e582a846f79"
PM = "0x8366a39cc670b4001a1121b8f6a443a643e40951"
QUOTER = "0x8dc178efb8111bb0973dd9d722ebeff267c98f94"
ROUTER = "0x6131b5fae19ea4f9d964eac0408e4408b66337b5"
EXEC = "0x8f10b468b06c6fd214b65f87778827f7d113f996"
SURPLUS = "0x1111110f0f73c0b2ef09ec012eae758b3e03a902"
ROLE = "0xe2f4eaae4a9751e85a3e4a7b9587827a877f29914755229b07a7b2da98285f70"
BASKETS = {"two": "0xfd4e8e1018bbc41b9f21a006e0410650b170c09d",  # bAMZNAAPL (AMZN+AAPL)
           "mag5": "0xca99aad9aad93475452179e035332cfd7d49a4aa"}  # LOT-Mag 5
PIPEDOG, PIPEDOG_POOL = "0x5cb6f181081301b44905f3ae15419112ecabd8a6", "0xb7f10f74b39291b9290b779978e19a7637c742d6"
WALLET, WALLET_POOL = "0x0339f5459fc690ac85f1782e15782a151b4a9e1b", "0x9501a20bedb8bea0798fe5d4c411f5e270965d49"
NVDA = "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec"
EOA = "0x4663000000000000000000000000000000004b4e"  # trader (kx.py)
OWNER = "0x46630000000000000000000000000000006f776e"  # fork-local hook / MQ owner (impersonated)
TREASURY = "0x4663000000000000000000000000000000747265"
MIN1, MAX1 = 4295128740, 1461446703485210103287273052203988822378723970341
# KYBER-PACK §5 recommended constants (B3): 2-stock 3.9M, Mag 5 <= $10k 5.9M, max fill 7.8M. MQ: none in §5.
GAS_CONST = {"two": 3_900_000, "mag5": 5_900_000, "mq": 7_800_000}
LEG_EXECUTED = None  # filled lazily (cast keccak)
OUT = None


def log(*a):
    print(f"[kd {time.strftime('%H:%M:%S')}]", *a, flush=True)


def sh(*a, check=True):
    r = subprocess.run(list(a), capture_output=True, text=True)
    if check and r.returncode:
        raise RuntimeError(f"{a[:3]} -> {r.stderr[-800:]}")
    return r.stdout.strip()


def rpc(method, params, url=ANVIL):
    assert url.startswith("http://127.0.0.1:") or method in ("eth_blockNumber", "eth_chainId"), "writes local only"
    body = json.dumps({"jsonrpc": "2.0", "id": 1, "method": method, "params": params}).encode()
    for k in range(3):
        try:
            r = json.load(urllib.request.urlopen(urllib.request.Request(url, body, {"content-type": "application/json"}), timeout=300))
            break
        except Exception as e:  # transient local socket errors
            if k == 2:
                raise
            time.sleep(1)
    if "historical state" in json.dumps(r):
        raise SystemExit("PRUNED: the public RPC no longer serves the fork block's state (exit 3 semantics) — re-run the session")
    return r


def calldata(sig, *args):
    return sh(CAST, "calldata", sig, *[str(a) for a in args])


def call(to, data, frm=EOA, gas=None):
    tx = {"from": frm, "to": to, "data": data}
    if gas:
        tx["gas"] = hex(gas)
    r = rpc("eth_call", [tx, "latest"])
    if "error" in r:
        e = r["error"]
        return None, (e.get("data") if isinstance(e.get("data"), str) else json.dumps(e))
    return r["result"], None


def _split_sig(sig):
    """'name(args)(rets)' -> ('name(args)', 'rets' or None); tuple-aware."""
    i = sig.index("(")
    depth = 0
    for j in range(i, len(sig)):
        depth += {"(": 1, ")": -1}.get(sig[j], 0)
        if depth == 0:
            break
    rest = sig[j + 1:]
    return sig[:j + 1], (rest[1:-1] if rest else None)


def view(to, sig, *args):
    fn, rt = _split_sig(sig)
    res, err = call(to, calldata(fn, *args))
    if err:
        raise RuntimeError(f"view {sig} reverted: {err[:300]}")
    return sh(CAST, "abi-decode", "--input", f"f({rt})", res).splitlines() if rt else res


def u(x):
    return int(x.split()[0])


def imp(a, eth=10**21):
    rpc("anvil_impersonateAccount", [a])
    rpc("anvil_setBalance", [a, hex(eth)])


def send(frm, to, data, gas=15_000_000):
    tx = {"from": frm, "data": data, "gas": hex(gas)}
    if to:
        tx["to"] = to
    r = rpc("eth_sendTransaction", [tx])
    if "error" in r:
        # anvil still mines a reverting tx only if it passes pre-checks; surface the error
        raise RuntimeError(f"send failed: {json.dumps(r['error'])[:600]}")
    h = r["result"]
    for _ in range(600):
        rc = rpc("eth_getTransactionReceipt", [h])["result"]
        if rc:
            return h, rc
        time.sleep(0.5)
    raise RuntimeError(f"no receipt for {h}")


def snap():
    return rpc("evm_snapshot", [])["result"]


def revert(s):
    assert rpc("evm_revert", [s])["result"] is True, "evm_revert failed"


def bal(t, a):
    return u(view(t, "balanceOf(address)(uint256)", a)[0])


def w(a):
    return "00" * 12 + a[2:].lower()


def h16(x):
    return x.to_bytes(16, "big").hex()


# ------------------------------------------------------------------------------------------------ anvil
def start_anvil(tag):
    if sh("lsof", "-nP", "-t", "-iTCP:8591", "-sTCP:LISTEN", check=False):
        raise SystemExit("port 8591 already has a listener — stop it first")
    head = int(sh(CAST, "block-number", "--rpc-url", PUBLIC))  # read only
    pin = head - 5
    lf = open(f"{OUT}/anvil.log", "w")
    p = subprocess.Popen([ANVILBIN, "--fork-url", PUBLIC, "--fork-block-number", str(pin), "--host", "127.0.0.1",
                          "--port", "8591", "--chain-id", "4663"], stdout=lf, stderr=lf,
                         start_new_session=True)  # outlives this script; stopped by `down`
    for _ in range(60):
        try:
            if rpc("eth_chainId", [])["result"] == hex(4663):
                break
        except Exception:
            time.sleep(1)
    rpc("anvil_mine", ["0x1"])  # else 'Excess blob gas not set'
    log(f"anvil pid {p.pid} fork pin {pin} (head {head}) chain 4663 on 127.0.0.1:8591")
    return p, pin


def stop_anvil(p):
    p.terminate()
    try:
        p.wait(10)
    except Exception:
        p.kill()
    left = sh("lsof", "-nP", "-t", "-iTCP:8591", "-sTCP:LISTEN", check=False)
    log("anvil stopped", "(port free)" if not left else f"(STILL LISTENING pid {left})")


# ------------------------------------------------------------------------------------------------ B1 deploy
def deploy(with_mq=True):
    t0 = time.time()
    env = dict(os.environ, KD_OWNER=OWNER, KD_TREASURY=TREASURY)
    r = subprocess.run([FORGE, "script", "script/kyber-delivery/V17ForkPlan.s.sol:V17ForkPlan", "--fork-url", ANVIL],
                       cwd=CONTRACTS, env=env, capture_output=True, text=True)
    line = [l for l in r.stdout.splitlines() if "KDPLAN " in l]
    if r.returncode or not line:
        raise RuntimeError("plan script failed: " + r.stdout[-1500:] + r.stderr[-1500:])
    plan = json.loads(line[0].split("KDPLAN ", 1)[1])
    log(f"plan ok ({time.time() - t0:.0f}s): hook {plan['hook']} hookMq {plan['hookMq']}")
    d = {"forkBlockAtPlan": plan["forkBlock"], "txs": {}}

    def rec(name, h, rc):
        st = rc["status"]
        d["txs"][name] = {"tx": h, "status": st, "gasUsed": int(rc["gasUsed"], 16)}
        assert st == "0x1", f"{name} failed {h}"

    for dep, code, at, nm in ((plan["legsDeployer"], plan["legsCode"], plan["legsAddr"], "NavJitLegs"),
                              (plan["legsMqDeployer"], plan["legsMqCode"], plan["legsMqAddr"], "NavJitLegsMq")):
        imp(dep)
        assert int(rpc("eth_getTransactionCount", [dep, "latest"])["result"], 16) == 0, f"{nm} deployer nonce != 0"
        h, rc = send(dep, None, code, 30_000_000)
        rec(nm, h, rc)
        assert rc["contractAddress"].lower() == at.lower(), f"{nm} landed at {rc['contractAddress']}"
    imp(OWNER)
    c2 = plan["create2"]
    for nm, salt, init, at in (("NavJitHookV17", plan["hookSalt"], plan["hookInit"], plan["hook"]),
                               ("MqRegistry", plan["mqRegistrySalt"], plan["mqRegistryInit"], plan["mqRegistry"]),
                               ("MqFactory", plan["mqFactorySalt"], plan["mqFactoryInit"], plan["mqFactory"]),
                               ("NavGuardMq", plan["navGuardMqSalt"], plan["navGuardMqInit"], plan["navGuardMq"]),
                               ("NavJitHookV17Mq", plan["hookMqSalt"], plan["hookMqInit"], plan["hookMq"])):
        h, rc = send(OWNER, c2, salt + init[2:], 30_000_000)
        rec(nm, h, rc)
        code = rpc("eth_getCode", [at, "latest"])["result"]
        assert len(code) > 2, f"{nm}: no code at {at}"
        d[nm] = at
    d["NavJitLegs"], d["NavJitLegsMq"] = plan["legsAddr"], plan["legsMqAddr"]
    reg, fac, hk, hmq = plan["mqRegistry"], plan["mqFactory"], plan["hook"], plan["hookMq"]
    rec("setWethToken.PIPEDOG", *send(OWNER, reg, calldata("setWethToken(address,address)", PIPEDOG, PIPEDOG_POOL)))
    rec("setWethToken.WALLET", *send(OWNER, reg, calldata("setWethToken(address,address)", WALLET, WALLET_POOL)))
    rec("MqFactory.setNavGuard", *send(OWNER, fac, calldata("setNavGuard(address)", plan["navGuardMq"])))
    rec("hookMq.setParams", *send(OWNER, hmq, calldata("setParams((uint16,uint16,uint16,uint16,int24))", "(25,200,600,300,20)")))
    venues = {}
    for nm, lot in BASKETS.items():
        rec(f"openVenue.{nm}", *send(OWNER, hk, calldata("openVenue(address)", lot), 15_000_000))
        venues[nm] = {"lot": lot, "hook": hk}
    if with_mq:
        # Explicit gas (eth_estimateGas under-estimates createBasket). Lot read back from chain, not simulated.
        rec("MqFactory.createBasket", *send(OWNER, fac, calldata("createBasket(address[],uint16[])",
                                                                 f"[{WALLET},{PIPEDOG},{NVDA}]", "[3000,3000,4000]"), 10_000_000))
        n = u(view(fac, "basketCount()(uint256)")[0])
        mq = view(fac, "baskets(uint256)(address)", n - 1)[0].lower()
        assert view(fac, "isBasket(address)(bool)", mq)[0] == "true"
        rec("openVenue.mq", *send(OWNER, hmq, calldata("openVenue(address)", mq), 15_000_000))
        venues["mq"] = {"lot": mq, "hook": hmq}
    # B1 assertions
    d["hook.registry"] = view(hk, "registry()(address)")[0].lower()
    d["hookMq.registry"] = view(hmq, "registry()(address)")[0].lower()
    assert d["hook.registry"] == STOCK_REGISTRY, d["hook.registry"]
    assert d["hookMq.registry"] == reg.lower(), d["hookMq.registry"]
    for nm, v in venues.items():
        key = view(v["hook"], "venueKey(address)((address,address,uint24,int24,address))", v["lot"])[0]
        a = re.findall(r"0x[0-9a-fA-F]{40}|-?\d+", key)
        enc = sh(CAST, "abi-encode", "f(address,address,uint24,int24,address)", *a)
        pid = sh(CAST, "keccak", enc)
        got = view(v["hook"], "venueOf(address)(bytes32)", v["lot"])[0]
        assert got == pid and got != "0x" + "00" * 32, (nm, got, pid)
        sp = view(PM, "extsload(bytes32)(bytes32)", slot0_slot(pid))[0]
        assert int(sp, 16) & ((1 << 160) - 1) > 0, f"{nm}: pool not initialized"
        v.update({"poolId": pid, "key": key, "symbol": view(v["lot"], "symbol()(string)")[0],
                  "constituents": view(v["lot"], "constituents()(address[])")[0]})
    d["venues"] = venues
    d["deploySeconds"] = round(time.time() - t0)
    json.dump(d, open(f"{OUT}/deploy.json", "w"), indent=1)
    log(f"B1 deployed in {d['deploySeconds']}s; registry asserts ok; venues", {k: v["poolId"] for k, v in venues.items()})
    return d


def slot0_slot(pid):
    # StateLibrary: keccak256(abi.encodePacked(poolId, POOLS_SLOT=6))
    return sh(CAST, "keccak", "0x" + pid[2:] + format(6, "064x"))




def main():
    global OUT
    cmd, OUT = sys.argv[1], os.path.abspath(sys.argv[2])
    os.makedirs(OUT, exist_ok=True)
    if cmd == "up":
        p, pin = start_anvil("parity")
        open(f"{OUT}/anvil.pid", "w").write(str(p.pid))
        try:
            d = deploy(with_mq=True)
        except BaseException:
            stop_anvil(p)
            raise
        d.update({"pin": pin, "anvil": ANVIL, "pm": PM, "quoter": QUOTER, "stockRegistry": STOCK_REGISTRY,
                  "owner": OWNER, "eoa": EOA})
        json.dump(d, open(f"{OUT}/deploy.json", "w"), indent=1)
        log("deploy.json written; anvil left running, pid", p.pid)
    elif cmd == "down":
        pid = open(f"{OUT}/anvil.pid").read().strip()
        listening = sh("lsof", "-nP", "-t", "-iTCP:8591", "-sTCP:LISTEN", check=False).split()
        if pid not in listening:
            raise SystemExit(f"pid {pid} is not the :8591 listener ({listening}); not killing anything")
        sh("kill", pid)
        log("killed anvil pid", pid)


if __name__ == "__main__":
    main()
