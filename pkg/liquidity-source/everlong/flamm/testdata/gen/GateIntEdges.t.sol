// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./GateEdges.t.sol";
import {Math} from "@openzeppelin/contracts/utils/math/Math.sol";

/// @dev OpenZeppelin (compat v4) Math.mulDiv, the copy every financing contract links, behind external entry points.
contract GateIntEdgesMathHarness {
    function mulDiv(uint256 x, uint256 y, uint256 d) external pure returns (uint256) {
        return Math.mulDiv(x, y, d);
    }

    function mulDivUp(uint256 x, uint256 y, uint256 d) external pure returns (uint256) {
        return Math.mulDiv(x, y, d, Math.Rounding.Up);
    }
}

/// @dev Integer edges: Math.mulDiv's zero-denominator and rounded-up overflow revert classes, and
///      FLAMMGateLib's int256 edges (wrapping casts, checked add / sub / negation at type(int256).min) plus the
///      checked epsilon shave. Rows reuse the GateEdges row format.
contract GateIntEdges is GateEdges {
    GateIntEdgesMathHarness internal mh;

    function _mdRes(bytes memory data) internal view returns (string memory) {
        (bool ok, bytes memory ret) = address(mh).staticcall(data);
        return _res(ok, ret);
    }

    function _mdRow(uint256 x, uint256 y, uint256 d) internal {
        string memory o = "{";
        o = add(o, "x", q(x));
        o = add(o, "y", q(y));
        o = add(o, "d", q(d));
        o = add(o, "down", _mdRes(abi.encodeCall(GateIntEdgesMathHarness.mulDiv, (x, y, d))));
        o = end(add(o, "up", _mdRes(abi.encodeCall(GateIntEdgesMathHarness.mulDivUp, (x, y, d)))));
        row(o);
    }

    function test_intEdgesMulDiv() public {
        mh = new GateIntEdgesMathHarness();
        uint256 max = type(uint256).max;
        // floor(x*y/d) == max with a non-zero remainder: the rounded-up `result += 1` overflows
        (bool ok, bytes memory ret) = address(mh).staticcall(
            abi.encodeCall(GateIntEdgesMathHarness.mulDivUp, ((uint256(1) << 255) + 1, max - 1, uint256(1) << 255))
        );
        assertFalse(ok);
        assertEq(ret, abi.encodeWithSignature("Panic(uint256)", uint256(0x11)));
        uint256[15] memory s = [
            uint256(0), 1, 2, 3, 1e18 - 1, 1e18, 3814697265625, type(uint128).max, uint256(1) << 128,
            (uint256(1) << 255) - 2, (uint256(1) << 255) - 1, uint256(1) << 255, (uint256(1) << 255) + 1, max - 1, max
        ];
        open("mm_muldiv_edges.json");
        for (uint256 i; i < 15; ++i) {
            for (uint256 j; j < 15; ++j) {
                for (uint256 k; k < 15; ++k) _mdRow(s[i], s[j], s[k]);
            }
        }
        for (uint256 seed; seed < 400; ++seed) {
            uint256 x = rbits(seed, 1, 256);
            uint256 y = rbits(seed, 2, 256);
            uint256 d = rnd(seed, 3) % 8 == 0 ? 0 : rbits(seed, 4, 256);
            _mdRow(x, y, d);
        }
        close();
    }

    function test_intEdgesGate() public {
        h = new VGateHarness();
        router = new VGateRouter();
        feed = new VGateFeed();
        mh = new GateIntEdgesMathHarness();
        uint256 max = type(uint256).max;
        uint256 half = uint256(1) << 255;
        uint256 fiveP18 = 3814697265625; // 5**18
        assertEq(fiveP18 << 18, 1e18);
        open("gate_int_edges.json");
        G memory g;

        // requiredPosted: mulDivUp(num, WAD, ltv*price) whose floor is exactly max with a remainder
        g = _base("up-overflow-requiredPosted", 1);
        g.debt[0] = Math.mulDiv(1e18 - 1, max, 1e18) + 1;
        g.scale[0] = 1;
        g.ltvArg = 1e18 - 1;
        g.price[0] = 1;
        {
            (bool ok, bytes memory ret) =
                address(mh).staticcall(abi.encodeCall(GateIntEdgesMathHarness.mulDivUp, (g.debt[0], 1e18, 1e18 - 1)));
            assertFalse(ok);
            assertEq(ret, abi.encodeWithSignature("Panic(uint256)", uint256(0x11)));
        }
        this.runCase(g);

        // context: mulDivUp(debt*scale, cross, WAD) whose floor is exactly max with a remainder
        g = _base("up-overflow-context", 1);
        g.scale[0] = 1;
        for (uint256 qq = 1e18 + 1; qq < 1e18 + 64; ++qq) {
            uint256 n18 = Math.mulDiv(1e18, max, qq) + 1;
            (bool ok, bytes memory ret) =
                address(mh).staticcall(abi.encodeCall(GateIntEdgesMathHarness.mulDivUp, (n18, qq, 1e18)));
            if (!ok && keccak256(ret) == keccak256(abi.encodeWithSignature("Panic(uint256)", uint256(0x11)))) {
                g.debt[0] = n18;
                g.cross[0] = qq;
                break;
            }
        }
        assertTrue(g.cross[0] != 1e18);
        this.runCase(g);

        // netL18: int256(2^255 - 1) - int256(max) = max_int - (-1) overflows
        g = _base("netL18-sub-overflow", 1);
        g.scale[0] = 1;
        g.debt[0] = half - 1;
        g.supplied[0] = max;
        this.runCase(g);

        // netL18 at exactly int256 max: netPW's `uint256(n) * WAD` overflows
        g = _base("netL18-max", 1);
        g.scale[0] = 1;
        g.debt[0] = half - 1;
        this.runCase(g);

        // netL18 = int256(2^255) = type(int256).min: netPW's `-n` reverts
        g = _base("netL18-min", 1);
        g.scale[0] = 1;
        g.debt[0] = half;
        this.runCase(g);

        // netL18 = int256(0) - int256(2^255) = 0 - min overflows
        g = _base("netL18-zero-minus-min", 1);
        g.scale[0] = 1;
        g.debt[0] = 0;
        g.supplied[0] = half;
        this.runCase(g);

        // netL18 = int256(2^256 - 1) - int256(2^255) = -1 - min = max_int
        g = _base("netL18-wrapped-minus-min", 1);
        g.scale[0] = 1;
        g.debt[0] = max;
        g.supplied[0] = half;
        this.runCase(g);

        // netPW positive side: ceilDiv lands in [2^255, 2^256) and the int256 cast wraps negative
        g = _base("netPW-pos-wrap", 2);
        g.scale[0] = 1;
        g.debt[0] = 6e58;
        g.price[0] = 1;
        this.runCase(g);

        // netPW positive side exactly 2^255: the cast is type(int256).min, headOf's `-n` reverts
        g = _base("netPW-pos-min", 1);
        g.scale[0] = 1;
        g.debt[0] = uint256(1) << 237;
        g.price[0] = fiveP18;
        this.runCase(g);

        // netPW surplus side: the floor lands in [2^255, 2^256), `-int256(q)` of a wrapped negative is positive
        g = _base("netPW-neg-wrap", 2);
        g.scale[0] = 1;
        g.debt[0] = 0;
        g.supplied[0] = 6e58;
        g.price[0] = 1;
        g.u0[0] = 1;
        this.runCase(g);

        // netPW surplus side exactly 2^255: `-int256(q)` of type(int256).min reverts
        g = _base("netPW-neg-min", 1);
        g.scale[0] = 1;
        g.debt[0] = 0;
        g.supplied[0] = uint256(1) << 237;
        g.price[0] = fiveP18;
        this.runCase(g);

        // roomWad at u = type(int256).min: `uint256(-u)` reverts
        g = _base("roomWad-min", 1);
        g.uArg = type(int256).min;
        this.runCase(g);

        // roomNative with an epsilon above WAD: the shave exceeds raw and the checked `-=` reverts
        g = _base("roomNative-eps-over-wad", 2);
        g.eps = 2e18;
        this.runCase(g);
        g = _base("roomNative-eps-wad", 2);
        g.eps = 1e18;
        this.runCase(g);

        // _readableU: frozen*scale >= 2^255 wraps negative, so the subtraction adds
        g = _base("readableU-wrap", 1);
        g.qAny[0] = true;
        g.qDebt[0] = max / 1e12 - 5;
        g.qColl[0] = 1;
        g.quarantined = true;
        this.runCase(g);

        // _readableU: frozen*scale == 2^255 is type(int256).min; a negative net absorbs it, a positive one overflows
        g = _base("readableU-min-surplus", 1);
        g.scale[0] = 1;
        g.debt[0] = 0;
        g.supplied[0] = 10;
        g.qAny[0] = true;
        g.qDebt[0] = half;
        g.u0[0] = -10;
        this.runCase(g);
        g = _base("readableU-min-debt", 1);
        g.scale[0] = 1;
        g.debt[0] = 5;
        g.qAny[0] = true;
        g.qDebt[0] = half;
        this.runCase(g);

        // assertEntryGate quarantined: u0 + int256(fDebt*scale) overflows at u0 = int256 max
        g = _base("entry-u0raw-overflow", 1);
        g.ltv = 0.05e18; // exposure above the bound, so the per-asset form runs
        g.qAny[0] = true;
        g.qDebt[0] = 1;
        g.qColl[0] = 1;
        g.quarantined = true;
        g.u0[0] = type(int256).max;
        this.runCase(g);

        // assertEntryGate quarantined: fDebt*scale wraps negative, u0raw <= 0 reverts LedgerBoundBreached
        g = _base("entry-u0raw-wrap", 1);
        g.ltv = 0.05e18; // exposure above the bound, so the per-asset form runs
        g.qAny[0] = true;
        g.qDebt[0] = max / 1e12 - 1;
        g.qColl[0] = 1;
        g.quarantined = true;
        this.runCase(g);

        // assertEntryGate quarantined: fDebt*scale == 2^255 added to a positive u0 stays in range
        g = _base("entry-u0raw-min", 1);
        g.ltv = 0.05e18; // exposure above the bound, so the per-asset form runs
        g.scale[0] = 1;
        g.debt[0] = 11_301_759e12;
        g.qAny[0] = true;
        g.qDebt[0] = half;
        g.qColl[0] = 1;
        g.quarantined = true;
        g.u0[0] = 1;
        this.runCase(g);
        close();
    }
}
