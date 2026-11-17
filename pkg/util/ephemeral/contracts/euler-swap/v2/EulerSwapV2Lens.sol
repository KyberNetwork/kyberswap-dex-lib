// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

import {ChainBlock} from "../../libraries/ChainBlock.sol";

interface IEulerSwapV2 {
    function isInstalled() external view returns (bool);
    function getReserves() external view;
    function getDynamicParams() external view;
}

interface IEVC {
    function getCollaterals(address account) external view returns (address[] memory);
    function getControllers(address account) external view returns (address[] memory);
    function isControllerEnabled(address account, address vault) external view returns (bool);
}

interface IEVault {
    function cash() external view;
    function debtOf(address account) external view;
    function maxDeposit(address account) external view;
    function caps() external view;
    function totalBorrows() external view;
    function totalAssets() external view;
    function totalSupply() external view;
    function balanceOf(address account) external view;
    function oracle() external view returns (address);
    function unitOfAccount() external view returns (address);
    function decimals() external view returns (uint8);
    function asset() external view returns (address);
    function LTVBorrow(address collateral) external view;
}

interface IEulerRouter {
    function getQuotes(uint256 inAmount, address base, address quote) external view;
}

/// @title EulerSwapV2Lens
/// @notice Deployless lens that reads every EulerSwap v2 pool it is given in one `eth_call`,
/// dependent reads included.
///
/// @dev Call with `to == null` and `data == creationCode ++ abi.encode(PoolInput[] pools)`; the
/// constructor reverts with `abi.encode(getPoolStates' return values)`.
///
/// The tracker reads a pool in up to four rounds, each depending on the last: the pool, its vaults
/// and the Euler account's controllers and collaterals; then the controller vault if it is not one
/// of the pool's own; then every collateral that is not a vault; then an oracle quote and an LTV for
/// every vault against every vault and collateral. Over RPC that is four round trips per pool.
///
/// This lens makes the same calls, in the same order, for every pool, and returns each call's raw
/// `(success, returnData)` grouped by round, exactly what `tryAggregate` would have returned. It
/// decodes only what it needs to build the next round (addresses, decimals) and leaves every value
/// the pool state is made of to the caller, which replays its own round construction over these
/// results. The business logic therefore stays in one place, the tracker; this contract only removes
/// the round trips.
///
/// Decoding follows go-ethereum's rules wherever the next round depends on it, so the calls built
/// here are the ones the tracker would build from the same data: an address takes the low 20 bytes
/// of a word of at least 32 bytes, a uint8 larger than 255 is a decode failure (and stays 0), and a
/// malformed address[] reads as empty. A failed or malformed read never reverts the whole call.
contract EulerSwapV2Lens {
    /// @notice One pool to read. `vaults` is the tracker's deduplicated vault list, in its order,
    /// with the asset and quote amount (10 ** decimals) of each.
    struct PoolInput {
        address pool;
        address eulerAccount;
        address evc;
        address[] vaults;
        address[] vaultAssets;
        uint256[] quoteAmounts;
    }

    /// @notice One call's outcome, as Multicall's tryAggregate reports it.
    struct Result {
        bool success;
        bytes returnData;
    }

    /// @notice A pool's calls, grouped by the tracker's rounds. A round the tracker would skip is
    /// empty.
    struct PoolReads {
        Result[] round1;
        Result[] round2;
        Result[] round3;
        Result[] round4;
    }

    /// @notice `vaults`, `vaultAssets` and `quoteAmounts` differ in length.
    error MismatchedVaultInput();

    /// @notice The call ran out of gas for the next read. Recording that read as an ordinary failure,
    /// the way tryAggregate would, returns complete-looking pools with values missing, and the more
    /// pools a call carries, the likelier it reaches the node's gas cap. Reverting instead tells the
    /// caller to read fewer pools per call.
    error GasExhausted();

    /// @notice Gas each read may use. Generous for a view call (a whole pool's reads, oracle quotes
    /// included, came to ~0.55M on Ethereum), and bounded so a callee that burns everything it is
    /// given (an INVALID opcode, a failing precompile) costs one failed read rather than the batch.
    uint256 internal constant READ_GAS = 2_000_000;

    /// @dev Headroom for copying a read's result back and moving on to the next.
    uint256 private constant READ_OVERHEAD = 20_000;

    constructor(PoolInput[] memory pools) payable {
        (uint256 blockNumber, PoolReads[] memory reads) = getPoolStates(pools);
        bytes memory returnData = abi.encode(blockNumber, reads);
        assembly ("memory-safe") {
            revert(add(returnData, 0x20), mload(returnData))
        }
    }

    /// @notice Reads every pool's rounds at the current block.
    /// @return blockNumber The block every read was made at.
    /// @return reads One entry per pool, in input order.
    function getPoolStates(PoolInput[] memory pools)
        public
        view
        returns (uint256 blockNumber, PoolReads[] memory reads)
    {
        blockNumber = ChainBlock.number();
        reads = new PoolReads[](pools.length);
        for (uint256 i; i < pools.length; ++i) {
            reads[i] = _readPool(pools[i]);
        }
    }

    /// @dev The vault list a pool ends up with after round 2, and what round 4 needs of each.
    struct Vaults {
        address[] addr;
        address[] asset;
        uint256[] quote;
        address[] oracle;
        address[] unit;
    }

    function _readPool(PoolInput memory p) private view returns (PoolReads memory out) {
        uint256 n = p.vaults.length;
        // Malformed input is the caller's bug, not one pool's state: fail the call.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(p.vaultAssets.length == n && p.quoteAmounts.length == n, MismatchedVaultInput());

        // Round 1: the pool, the account's controllers and collaterals, and eleven reads per vault.
        out.round1 = new Result[](5 + 11 * n);
        out.round1[0] = _read(p.pool, abi.encodeCall(IEulerSwapV2.isInstalled, ()));
        out.round1[1] = _read(p.evc, abi.encodeCall(IEVC.getCollaterals, (p.eulerAccount)));
        out.round1[2] = _read(p.evc, abi.encodeCall(IEVC.getControllers, (p.eulerAccount)));
        out.round1[3] = _read(p.pool, abi.encodeCall(IEulerSwapV2.getReserves, ()));
        out.round1[4] = _read(p.pool, abi.encodeCall(IEulerSwapV2.getDynamicParams, ()));
        for (uint256 i; i < n; ++i) {
            address v = p.vaults[i];
            uint256 k = 5 + 11 * i;
            out.round1[k] = _read(v, abi.encodeCall(IEVault.cash, ()));
            out.round1[k + 1] = _read(v, abi.encodeCall(IEVault.debtOf, (p.eulerAccount)));
            out.round1[k + 2] = _read(v, abi.encodeCall(IEVault.maxDeposit, (p.eulerAccount)));
            out.round1[k + 3] = _read(v, abi.encodeCall(IEVault.caps, ()));
            out.round1[k + 4] = _read(v, abi.encodeCall(IEVault.totalBorrows, ()));
            out.round1[k + 5] = _read(v, abi.encodeCall(IEVault.totalAssets, ()));
            out.round1[k + 6] = _read(v, abi.encodeCall(IEVault.totalSupply, ()));
            out.round1[k + 7] = _read(v, abi.encodeCall(IEVault.balanceOf, (p.eulerAccount)));
            out.round1[k + 8] = _read(v, abi.encodeCall(IEVault.oracle, ()));
            out.round1[k + 9] = _read(v, abi.encodeCall(IEVault.unitOfAccount, ()));
            out.round1[k + 10] = _read(p.evc, abi.encodeCall(IEVC.isControllerEnabled, (p.eulerAccount, v)));
        }

        address[] memory collaterals = _addressArray(out.round1[1]);
        address[] memory controllers = _addressArray(out.round1[2]);

        // Round 2: the controller, when it is not one of the pool's vaults, joins the list.
        bool addController = controllers.length > 0 && !_contains(p.vaults, n, controllers[0]);
        Vaults memory vs = _vaults(p, out.round1, n + (addController ? 1 : 0));
        if (addController) {
            address c = controllers[0];
            out.round2 = new Result[](5);
            out.round2[0] = _read(c, abi.encodeCall(IEVault.debtOf, (p.eulerAccount)));
            out.round2[1] = _read(c, abi.encodeCall(IEVault.decimals, ()));
            out.round2[2] = _read(c, abi.encodeCall(IEVault.asset, ()));
            out.round2[3] = _read(c, abi.encodeCall(IEVault.oracle, ()));
            out.round2[4] = _read(c, abi.encodeCall(IEVault.unitOfAccount, ()));
            vs.addr[n] = c;
            vs.asset[n] = _address(out.round2[2]);
            vs.quote[n] = _tenPow(_uint8(out.round2[1]));
            vs.oracle[n] = _address(out.round2[3]);
            vs.unit[n] = _address(out.round2[4]);
        }

        // Round 3: five reads per collateral that is not already a vault.
        uint256 total = vs.addr.length;
        bool[] memory isVault = new bool[](collaterals.length);
        uint256 external_ = 0;
        for (uint256 j; j < collaterals.length; ++j) {
            isVault[j] = _contains(vs.addr, total, collaterals[j]);
            if (!isVault[j]) ++external_;
        }
        out.round3 = new Result[](5 * external_);
        uint256[] memory collatQuote = new uint256[](collaterals.length);
        address[] memory collatAsset = new address[](collaterals.length);
        uint256 r = 0;
        for (uint256 j; j < collaterals.length; ++j) {
            if (isVault[j]) continue;
            address c = collaterals[j];
            out.round3[r] = _read(c, abi.encodeCall(IEVault.balanceOf, (p.eulerAccount)));
            out.round3[r + 1] = _read(c, abi.encodeCall(IEVault.totalAssets, ()));
            out.round3[r + 2] = _read(c, abi.encodeCall(IEVault.totalSupply, ()));
            out.round3[r + 3] = _read(c, abi.encodeCall(IEVault.decimals, ()));
            out.round3[r + 4] = _read(c, abi.encodeCall(IEVault.asset, ()));
            collatQuote[j] = _tenPow(_uint8(out.round3[r + 3]));
            collatAsset[j] = _address(out.round3[r + 4]);
            r += 5;
        }

        // Round 4: for every vault with an oracle, a quote and an LTV against every vault, then
        // against every collateral that is not a vault.
        out.round4 = _round4(vs, total, collaterals, isVault, collatQuote, collatAsset);
    }

    function _round4(
        Vaults memory vs,
        uint256 total,
        address[] memory collaterals,
        bool[] memory isVault,
        uint256[] memory collatQuote,
        address[] memory collatAsset
    ) private view returns (Result[] memory out) {
        uint256 count = 0;
        for (uint256 i; i < total; ++i) {
            if (vs.oracle[i] == address(0)) continue;
            count += total + (total - 1);
            for (uint256 j; j < collaterals.length; ++j) {
                if (!isVault[j]) count += 2;
            }
        }

        out = new Result[](count);
        uint256 r = 0;
        for (uint256 i; i < total; ++i) {
            address oracle = vs.oracle[i];
            if (oracle == address(0)) continue;
            for (uint256 j; j < total; ++j) {
                out[r++] = _read(oracle, abi.encodeCall(IEulerRouter.getQuotes, (vs.quote[j], vs.asset[j], vs.unit[i])));
                if (i != j) out[r++] = _read(vs.addr[i], abi.encodeCall(IEVault.LTVBorrow, (vs.addr[j])));
            }
            for (uint256 j; j < collaterals.length; ++j) {
                if (isVault[j]) continue;
                out[r++] =
                    _read(oracle, abi.encodeCall(IEulerRouter.getQuotes, (collatQuote[j], collatAsset[j], vs.unit[i])));
                out[r++] = _read(vs.addr[i], abi.encodeCall(IEVault.LTVBorrow, (collaterals[j])));
            }
        }
    }

    /// @dev The pool's own vaults, with their oracle and unit of account from round 1, in a list
    /// sized for the controller to be appended.
    function _vaults(PoolInput memory p, Result[] memory round1, uint256 size) private pure returns (Vaults memory vs) {
        vs.addr = new address[](size);
        vs.asset = new address[](size);
        vs.quote = new uint256[](size);
        vs.oracle = new address[](size);
        vs.unit = new address[](size);
        for (uint256 i; i < p.vaults.length; ++i) {
            vs.addr[i] = p.vaults[i];
            vs.asset[i] = p.vaultAssets[i];
            vs.quote[i] = p.quoteAmounts[i];
            vs.oracle[i] = _address(round1[5 + 11 * i + 8]);
            vs.unit[i] = _address(round1[5 + 11 * i + 9]);
        }
    }

    function _read(address target, bytes memory data) private view returns (Result memory r) {
        // Start a read only with its whole budget in hand. Running short means the batch needs more
        // gas than the call was given: revert, so the caller reads fewer pools per call, rather than
        // record reads that failed for want of gas as if their callees had refused them.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(gasleft() >= READ_GAS + READ_GAS / 63 + READ_OVERHEAD, GasExhausted());
        // The loop is the batch, one read per call the tracker would make. The callees are the pool's
        // own vaults, EVC and oracles: one that returns a huge reply runs this call out of gas, which
        // the caller handles as it handles any call too large for the node.
        // forge-lint: disable-next-line(calls-loop, return-bomb)
        (r.success, r.returnData) = target.staticcall{gas: READ_GAS}(data);
    }

    function _contains(address[] memory list, uint256 len, address a) private pure returns (bool) {
        for (uint256 i; i < len; ++i) {
            if (list[i] == a) return true;
        }
        return false;
    }

    /// @dev go-ethereum: the low 20 bytes of the first word; zero when the call failed or returned
    /// less than a word.
    function _address(Result memory r) private pure returns (address a) {
        if (!r.success || r.returnData.length < 32) return address(0);
        bytes memory d = r.returnData;
        assembly ("memory-safe") {
            // Clean the upper 96 bits: Solidity assumes an address is already clean, and compares
            // and encodes it as the full word.
            a := and(mload(add(d, 0x20)), 0xffffffffffffffffffffffffffffffffffffffff)
        }
    }

    /// @dev go-ethereum rejects a uint8 word above 255, leaving the output at zero.
    function _uint8(Result memory r) private pure returns (uint8) {
        if (!r.success || r.returnData.length < 32) return 0;
        uint256 w;
        bytes memory d = r.returnData;
        assembly ("memory-safe") {
            w := mload(add(d, 0x20))
        }
        // forge-lint: disable-next-line(unsafe-typecast): w was just checked to fit.
        return w > type(uint8).max ? 0 : uint8(w);
    }

    /// @dev 10 ** d. Above 10 ** 77 a uint256 overflows, where the tracker's big.Int would not; no
    /// real token reports decimals anywhere near that, so saturate rather than revert the batch.
    function _tenPow(uint8 d) private pure returns (uint256) {
        if (d > 77) return type(uint256).max;
        return 10 ** uint256(d);
    }

    /// @dev A returned address[], decoded with go-ethereum's bounds checks; empty when the call
    /// failed or the encoding does not hold together.
    function _addressArray(Result memory r) private pure returns (address[] memory out) {
        bytes memory d = r.returnData;
        if (!r.success || d.length < 64) return out;
        uint256 offset;
        assembly ("memory-safe") {
            offset := mload(add(d, 0x20))
        }
        if (offset > d.length - 32) return out;
        uint256 len;
        assembly ("memory-safe") {
            len := mload(add(add(d, 0x20), offset))
        }
        if (len > (d.length - offset - 32) / 32) return out;
        out = new address[](len);
        for (uint256 i; i < len; ++i) {
            address a;
            assembly ("memory-safe") {
                a := and(
                    mload(add(add(add(d, 0x40), offset), mul(i, 0x20))),
                    0xffffffffffffffffffffffffffffffffffffffff
                )
            }
            out[i] = a;
        }
    }
}
