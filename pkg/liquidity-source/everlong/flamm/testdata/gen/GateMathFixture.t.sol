// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";
import {FLAMMGateLib} from "src/core/flamm/FLAMMGateLib.sol";
import {FLAMMStore} from "src/core/flamm/FLAMMStore.sol";
import {IMMRouter} from "src/interfaces/core/mm/IMMRouter.sol";
import {IPriceFeed} from "src/interfaces/core/IPriceFeed.sol";
import {PoolContext} from "src/interfaces/core/flamm/IFLAMMHooks.sol";

/// @dev Answers the four Router reads FLAMMGateLib makes, from values the generator sets.
contract GateMockRouter {
    uint256[] internal _coll;
    uint256[] internal _sup;
    uint256[] internal _debt;
    uint256 internal _posted;
    bool[] internal _qAny;
    uint256[] internal _qDebt;
    uint256[] internal _qColl;

    function set(uint256[] memory sup, uint256[] memory debt, uint256 posted) external {
        _sup = sup;
        _debt = debt;
        _posted = posted;
        delete _coll;
        for (uint256 i; i < sup.length; ++i) _coll.push(0);
    }

    function setQuarantine(bool[] memory any, uint256[] memory d, uint256[] memory c) external {
        _qAny = any;
        _qDebt = d;
        _qColl = c;
    }

    function positions(address) external view returns (uint256[] memory, uint256[] memory, uint256[] memory, uint256) {
        return (_coll, _sup, _debt, _posted);
    }

    function position(address, uint8 idx) external view returns (uint256, uint256, uint256) {
        return (0, _sup[idx], _debt[idx]);
    }

    function quarantine(address, uint8 idx) external view returns (bool, uint256, uint256) {
        if (idx >= _qAny.length) return (false, 0, 0);
        return (_qAny[idx], _qDebt[idx], _qColl[idx]);
    }
}

/// @dev peekCross/peekUsd keyed by token.
contract GateMockFeed {
    mapping(address => uint256) public cross;
    mapping(address => uint256) public usd;

    function set(address token, uint256 c, uint256 u) external {
        cross[token] = c;
        usd[token] = u;
    }

    function peekCross(address, address token) external view returns (bool, uint256, uint48) {
        uint256 c = cross[token];
        return (c != 0, c, uint48(block.timestamp));
    }

    function peekUsd(address token) external view returns (bool, uint256, uint48) {
        uint256 u = usd[token];
        return (u != 0, u, uint48(block.timestamp));
    }
}

/// @dev Hosts a FLAMMStore.S at the ERC-7201 slot so the storage-taking gate functions run as they do in the pool.
contract GateHarness {
    function configure(
        address router,
        address feed,
        address[] memory tokens,
        uint256[] memory scales,
        uint256[] memory liquids,
        uint256 physical,
        uint64 ltv,
        uint64 phi,
        uint64 eps
    ) external {
        FLAMMStore.S storage $ = FLAMMStore.s();
        $.router = IMMRouter(router);
        $.priceFeed = IPriceFeed(feed);
        $.poolAsset = address(0xB7C);
        delete $.loans;
        for (uint256 i; i < tokens.length; ++i) {
            FLAMMStore.LoanCfg storage c = $.loans.push();
            c.token = tokens[i];
            c.scale = scales[i];
            c.liquid = liquids[i];
        }
        $.physicalPoolAsset = physical;
        $.ltvWad = ltv;
        $.phiWad = phi;
        $.roomEpsilonWad = eps;
    }

    function roomNative(FLAMMGateLib.Book memory b, uint256 idx, uint256 u) external view returns (uint256) {
        return FLAMMGateLib.roomNative(FLAMMStore.s(), b, idx, u);
    }

    function priced() external view returns (FLAMMGateLib.Book memory) {
        return FLAMMGateLib.priced(FLAMMStore.s());
    }

    function assertGate() external view {
        FLAMMGateLib.assertGate(FLAMMStore.s());
    }

    function anchor() external view returns (int256[] memory, uint256, bool) {
        return FLAMMGateLib.anchor(FLAMMStore.s());
    }

    function assertEntryGate(int256[] memory u0, uint256 g0, bool q) external view {
        FLAMMGateLib.assertEntryGate(FLAMMStore.s(), u0, g0, q);
    }

    function assertExitNotWorsened(int256[] memory u0, uint256 g0) external view {
        FLAMMGateLib.assertExitNotWorsened(FLAMMStore.s(), u0, g0);
    }

    function netPW(FLAMMGateLib.Leg memory L) external pure returns (int256) {
        return FLAMMGateLib.netPW(L);
    }

    function exposurePW(FLAMMGateLib.Book memory b) external pure returns (uint256) {
        return FLAMMGateLib.exposurePW(b);
    }

    function headOf(FLAMMGateLib.Book memory b, uint256 idx, uint256 u, uint256 ltv) external pure returns (uint256) {
        return FLAMMGateLib.headOf(b, idx, u, ltv);
    }

    function requiredPostedAll(FLAMMGateLib.Book memory b, uint256 ltv) external pure returns (uint256) {
        return FLAMMGateLib.requiredPostedAll(b, ltv);
    }

    function navAt(FLAMMGateLib.Book memory b) external pure returns (uint256) {
        return FLAMMGateLib.navAt(b);
    }

    function context(FLAMMGateLib.Book memory b, uint256 p, uint48 ts, uint256 supply)
        external
        pure
        returns (PoolContext memory)
    {
        return FLAMMGateLib.context(b, p, ts, supply);
    }

    function roomWad(int256 u, uint256 g, uint256 p, uint256 ltv, uint256 phi) external pure returns (uint256) {
        return FLAMMGateLib.roomWad(u, g, p, ltv, phi);
    }

    function structuralDistWad(uint256 ltv, uint256 lltv) external pure returns (uint256) {
        return FLAMMGateLib.structuralDistWad(ltv, lltv);
    }
}

/// @notice Gate fixture generator: FLAMMGateLib at commit 80abd43 over a seeded grid of books, dials and
///         quarantine frames. Writes pkg/liquidity-source/everlong/flamm/testdata/gate_math.json.
contract GateMathFixture is Test {
    uint256 internal constant WAD = 1e18;
    GateHarness h;
    GateMockRouter r;
    GateMockFeed f;
    uint256 internal seed;

    function _rand(uint256 salt) internal returns (uint256) {
        seed = uint256(keccak256(abi.encode(seed, salt)));
        return seed;
    }

    function _u(uint256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _i(int256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _b(bool v) internal pure returns (string memory) {
        return v ? "true" : "false";
    }

    function _err(bytes memory e) internal pure returns (string memory) {
        return string.concat('"', vm.toString(e), '"');
    }

    function _amount(uint256 salt, uint256 scaleNative) internal returns (uint256) {
        uint256 x = _rand(salt);
        uint256 kind = x % 8;
        if (kind == 0) return 0;
        if (kind == 1) return (x >> 8) % 1000;
        return (x >> 16) % (scaleNative * (1 + ((x >> 64) % 1000)));
    }

    function setUp() public {
        h = new GateHarness();
        r = new GateMockRouter();
        f = new GateMockFeed();
        seed = 0xF1A;
    }

    struct Case {
        uint256 n;
        uint256[] scales;
        uint256[] liquid;
        uint256[] sup;
        uint256[] debt;
        uint256[] price;
        uint256[] crossW;
        address[] tokens;
        uint256 physical;
        uint256 posted;
        uint64 ltv;
        uint64 phi;
        uint64 eps;
        bool[] qAny;
        uint256[] qDebt;
        uint256[] qColl;
    }

    function _case(uint256 k) internal returns (Case memory c) {
        c.n = 1 + (_rand(k) % 3);
        c.scales = new uint256[](c.n);
        c.liquid = new uint256[](c.n);
        c.sup = new uint256[](c.n);
        c.debt = new uint256[](c.n);
        c.price = new uint256[](c.n);
        c.crossW = new uint256[](c.n);
        c.tokens = new address[](c.n);
        c.qAny = new bool[](c.n);
        c.qDebt = new uint256[](c.n);
        c.qColl = new uint256[](c.n);
        for (uint256 i; i < c.n; ++i) {
            uint256 dec = [uint256(6), 18, 8][_rand(i) % 3];
            c.scales[i] = 10 ** (18 - dec);
            uint256 unit = 10 ** dec;
            c.liquid[i] = _amount(1, unit * 10);
            c.sup[i] = _amount(2, unit * 100);
            c.debt[i] = _amount(3, unit * 200);
            // cross: L18 per poolAsset base unit, around a BTC-like 8-decimal pool asset
            uint256 pk = _rand(4) % 10;
            c.price[i] = pk == 0 ? 0 : (5e14 + (_rand(5) % 5e14));
            if (pk == 1) c.price[i] = 1 + (_rand(6) % 1e6);
            c.crossW[i] = i == 0 ? WAD : ((_rand(7) % 7 == 0) ? 0 : 5e17 + (_rand(8) % 3e18));
            c.tokens[i] = address(uint160(0x1000 + i));
            if (_rand(9) % 6 == 0) {
                c.qAny[i] = true;
                c.qDebt[i] = c.debt[i] == 0 ? 0 : (c.debt[i] * (1 + _rand(10) % 5)) / 6;
                c.qColl[i] = _amount(11, 1e8);
            }
        }
        c.physical = _amount(12, 1e8 * 10);
        c.posted = _amount(13, 1e8 * 10);
        c.ltv = uint64(1e17 + (_rand(14) % 8e17));
        c.phi = uint64(5e17 + (_rand(15) % 5e17 + 1));
        c.eps = uint64(_rand(16) % 1e17);
        if (k % 5 == 0) c.phi = 1e18;
    }

    function _book(Case memory c) internal pure returns (FLAMMGateLib.Book memory b) {
        b.physical = c.physical;
        b.posted = c.posted;
        b.legs = new FLAMMGateLib.Leg[](c.n);
        for (uint256 i; i < c.n; ++i) {
            b.legs[i] = FLAMMGateLib.Leg({
                liquid: c.liquid[i],
                supplied: c.sup[i],
                debt: c.debt[i],
                scale: c.scales[i],
                priceWad: c.price[i],
                crossWad: c.crossW[i]
            });
        }
    }

    function _uarr(uint256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", _u(a[i]));
        s = string.concat(s, "]");
    }

    function _iarr(int256[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", _i(a[i]));
        s = string.concat(s, "]");
    }

    function _barr(bool[] memory a) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < a.length; ++i) s = string.concat(s, i == 0 ? "" : ",", _b(a[i]));
        s = string.concat(s, "]");
    }

    function _caseJson(Case memory c) internal pure returns (string memory) {
        return string.concat(
            '{"scale":', _uarr(c.scales), ',"liquid":', _uarr(c.liquid), ',"supplied":', _uarr(c.sup), ',"debt":',
            _uarr(c.debt), ',"priceWad":', _uarr(c.price), ',"crossWad":', _uarr(c.crossW), ',"physical":',
            _u(c.physical), ',"posted":', _u(c.posted), ',"ltvWad":', _u(c.ltv), ',"phiWad":', _u(c.phi),
            ',"roomEpsilonWad":', _u(c.eps), ',"qAny":', _barr(c.qAny), ',"qDebt":', _uarr(c.qDebt), ',"qColl":',
            _uarr(c.qColl), "}"
        );
    }

    function _pureJson(Case memory c) internal view returns (string memory s) {
        FLAMMGateLib.Book memory b = _book(c);
        s = '"netPW":[';
        for (uint256 i; i < c.n; ++i) {
            try h.netPW(b.legs[i]) returns (int256 v) {
                s = string.concat(s, i == 0 ? "" : ",", '{"v":', _i(v), "}");
            } catch (bytes memory e) {
                s = string.concat(s, i == 0 ? "" : ",", '{"err":', _err(e), "}");
            }
        }
        s = string.concat(s, "]");
        uint256 u;
        bool uok;
        try h.exposurePW(b) returns (uint256 v) {
            (u, uok) = (v, true);
            s = string.concat(s, ',"exposurePW":{"v":', _u(v), "}");
        } catch (bytes memory e) {
            s = string.concat(s, ',"exposurePW":{"err":', _err(e), "}");
        }
        s = string.concat(s, ',"headOf":[');
        for (uint256 i; i < c.n; ++i) {
            try h.headOf(b, i, u, c.ltv) returns (uint256 v) {
                s = string.concat(s, i == 0 ? "" : ",", '{"v":', _u(v), "}");
            } catch (bytes memory e) {
                s = string.concat(s, i == 0 ? "" : ",", '{"err":', _err(e), "}");
            }
        }
        s = string.concat(s, '],"roomNative":[');
        for (uint256 i; i < c.n; ++i) {
            try h.roomNative(b, i, u) returns (uint256 v) {
                s = string.concat(s, i == 0 ? "" : ",", '{"v":', _u(v), "}");
            } catch (bytes memory e) {
                s = string.concat(s, i == 0 ? "" : ",", '{"err":', _err(e), "}");
            }
        }
        s = string.concat(s, "]");
        try h.requiredPostedAll(b, c.ltv) returns (uint256 v) {
            s = string.concat(s, ',"requiredPostedAll":{"v":', _u(v), "}");
        } catch (bytes memory e) {
            s = string.concat(s, ',"requiredPostedAll":{"err":', _err(e), "}");
        }
        try h.navAt(b) returns (uint256 v) {
            s = string.concat(s, ',"navAt":{"v":', _u(v), "}");
        } catch (bytes memory e) {
            s = string.concat(s, ',"navAt":{"err":', _err(e), "}");
        }
        try h.context(b, c.price[0], 1234, 5e18) returns (PoolContext memory x) {
            s = string.concat(
                s, ',"context":{"liquid":', _u(x.liquidLoanAsset), ',"supplied":', _u(x.suppliedLoanAsset), ',"debt":',
                _u(x.debtLoanAsset), ',"loanCount":', _u(x.loanCount), "}"
            );
        } catch (bytes memory e) {
            s = string.concat(s, ',"context":{"err":', _err(e), "}");
        }
        s = string.concat(s, ',"structuralDistWad":', _u(h.structuralDistWad(c.ltv, (c.ltv + c.phi) / 2 + 1e17)));
        s = string.concat(s, ',"roomWadNeg":', _u(h.roomWad(-int256(c.physical * 1e10), c.physical, 7e14, c.ltv, c.phi)));
    }

    function _load(Case memory c, uint256 postedAdj, int256[] memory debtAdj) internal {
        uint256[] memory d = new uint256[](c.n);
        for (uint256 i; i < c.n; ++i) {
            int256 x = int256(c.debt[i]) + debtAdj[i];
            d[i] = x < 0 ? 0 : uint256(x);
        }
        r.set(c.sup, d, c.posted + postedAdj);
        r.setQuarantine(c.qAny, c.qDebt, c.qColl);
        for (uint256 i; i < c.n; ++i) {
            f.set(c.tokens[i], c.price[i], c.crossW[i] == 0 ? 0 : c.crossW[i]);
        }
        h.configure(address(r), address(f), c.tokens, c.scales, c.liquid, c.physical, c.ltv, c.phi, c.eps);
    }

    function _poolJson(Case memory c, uint256 k) internal returns (string memory s) {
        int256[] memory zero = new int256[](c.n);
        _load(c, 0, zero);
        try h.assertGate() {
            s = '"assertGate":{"ok":true}';
        } catch (bytes memory e) {
            s = string.concat('"assertGate":{"err":', _err(e), "}");
        }
        (int256[] memory u0, uint256 g0, bool q0) = h.anchor();
        s = string.concat(s, ',"anchor":{"u0":', _iarr(u0), ',"gross0":', _u(g0), ',"quarantined":', _b(q0), "}");
        // a post-flow book: debt moved per asset, posted moved, physical moved
        int256[] memory dAdj = new int256[](c.n);
        for (uint256 i; i < c.n; ++i) {
            uint256 x = _rand(100 + k + i);
            int256 mag = int256((x >> 8) % (1 + c.debt[i] / 2 + 1e6));
            dAdj[i] = x % 2 == 0 ? mag : -mag;
        }
        uint256 postedAdj = _rand(200 + k) % (1 + c.posted / 4);
        uint256 physical0 = c.physical;
        c.physical = physical0 * (50 + (_rand(300 + k) % 100)) / 100;
        _load(c, postedAdj, dAdj);
        uint256 physical1 = c.physical;
        c.physical = physical0;
        try h.assertEntryGate(u0, g0, q0) {
            s = string.concat(s, ',"entry":{"ok":true}');
        } catch (bytes memory e) {
            s = string.concat(s, ',"entry":{"err":', _err(e), "}");
        }
        try h.assertExitNotWorsened(u0, g0) {
            s = string.concat(s, ',"exit":{"ok":true}');
        } catch (bytes memory e) {
            s = string.concat(s, ',"exit":{"err":', _err(e), "}");
        }
        s = string.concat(
            s, ',"post":{"physical":', _u(physical1), ',"postedAdj":', _u(postedAdj), ',"debtAdj":', _iarr(dAdj), "}"
        );
    }

    function test_gateFixture() public {
        vm.pauseGasMetering();
        string memory path = "kyber-out/gate_math.json";
        vm.writeFile(path, '{"cases":[\n');
        uint256 N = 400;
        for (uint256 k; k < N; ++k) {
            Case memory c = _case(k);
            string memory inJson = _caseJson(c);
            _load(c, 0, new int256[](c.n));
            string memory pureJson = _pureJson(c);
            string memory poolJson = _poolJson(c, k);
            string memory row = string.concat('{"in":', inJson, ",", pureJson, ",", poolJson, "}");
            vm.writeLine(path, string.concat(k == 0 ? "" : ",", row));
        }
        vm.writeLine(path, "]}");
    }
}
