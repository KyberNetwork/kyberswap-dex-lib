// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {CollRebalancerMathCopy, LevCurve} from "./CollRebalancerMathCopy.sol";
import {Mul512} from "src/libraries/math/Mul512.sol";

/// @notice Exposes every internal/private function of the verbatim CollRebalancerMath copy.
///         Multi-argument entrypoints take packed arrays to stay under the legacy stack limit.
contract LevCurveEdgesHarness {
    function _c() private pure returns (LevCurve memory) {
        return CollRebalancerMathCopy.frozenCurve();
    }

    function ab(uint256[4] calldata a) external pure returns (uint256, uint256) {
        return CollRebalancerMathCopy.anchorAndBase(_c(), a[0], a[1], a[2], a[3]);
    }

    function lq(uint256[6] calldata a) external pure returns (uint256, uint256, uint256) {
        return CollRebalancerMathCopy.leverageQuote(_c(), a[0], a[1], a[2], a[3], a[4], a[5]);
    }

    function dq(uint256[6] calldata a) external pure returns (uint256, uint256, uint256) {
        return CollRebalancerMathCopy.deleverageQuote(_c(), a[0], a[1], a[2], a[3], a[4], a[5]);
    }

    function ss(uint256[5] calldata a) external pure returns (bool) {
        return CollRebalancerMathCopy.isStateSafe(_c(), a[0], a[1], a[2], a[3], a[4]);
    }

    function mv(uint256 collateral, uint256 price) external pure returns (bool, uint256) {
        return CollRebalancerMathCopy._markedValue(collateral, price);
    }

    function sa(uint256 cv, uint256 debt, uint256 rWad) external pure returns (bool, uint256, uint256, bool) {
        return CollRebalancerMathCopy._strictAnchor(_c(), cv, debt, rWad);
    }

    function hla(uint256 cv, uint256 debt) external pure returns (uint256) {
        return CollRebalancerMathCopy._halfLawAnchor(cv, debt);
    }

    function ric(uint256 cv, uint256 debt, uint256 anchor) external pure returns (bool) {
        return CollRebalancerMathCopy._rootIntervalContains(cv, debt, anchor);
    }

    function cvr(uint256 anchor, uint256 debt) external pure returns (bool, uint256) {
        return CollRebalancerMathCopy._cvRequiredOnAnchor(_c(), anchor, debt);
    }

    function dcap(uint256 anchor, uint256 cv) external pure returns (bool, uint256) {
        return CollRebalancerMathCopy._debtCapOnAnchor(_c(), anchor, cv);
    }

    function rs(uint256 cv, uint256 debt, uint256 rWad) external pure returns (CollRebalancerMathCopy.RecoveryState memory) {
        return CollRebalancerMathCopy._recoveryState(_c(), cv, debt, rWad);
    }

    function rdy(uint256 wallCv, uint256 y) external pure returns (uint256) {
        return CollRebalancerMathCopy._recoveryDebtAtY(_c(), wallCv, y);
    }

    function dsp(uint256 cv, uint256 debt, uint256 posted) external pure returns (uint256) {
        return CollRebalancerMathCopy._deleverageSpread(cv, debt, posted);
    }

    function pr(uint256[7] calldata a) external pure returns (uint256, uint256) {
        return CollRebalancerMathCopy._deleverageProRata(_c(), a[0], a[1], a[2], a[3], a[4], a[5], a[6]);
    }

    function rdl(uint256[7] calldata a) external pure returns (uint256, uint256, uint256) {
        return CollRebalancerMathCopy._recoveryDeleverage(_c(), a[0], a[1], a[2], a[3], a[4], a[5], a[6]);
    }

    function psa(uint256[6] calldata a) external pure returns (bool) {
        return CollRebalancerMathCopy._postStrictAnchorAccepted(_c(), a[0], a[1], a[2], a[3], a[4], a[5]);
    }

    function paa(uint256[6] calldata a) external pure returns (bool) {
        return CollRebalancerMathCopy._postAnyAnchorAccepted(_c(), a[0], a[1], a[2], a[3], a[4], a[5]);
    }

    function abe(uint256 cv, uint256 debt, uint256 rWad) external pure returns (uint256) {
        return CollRebalancerMathCopy.anchorBestEffort(_c(), cv, debt, rWad);
    }

    function phi(uint256 h) external pure returns (uint256) {
        return CollRebalancerMathCopy._phiWad(_c(), h);
    }

    function dn(uint256 h) external pure returns (uint256) {
        return CollRebalancerMathCopy._debtNormWad(_c(), h);
    }

    function lerp(uint256 a, uint256 b, uint256 x) external pure returns (uint256) {
        return CollRebalancerMathCopy._lerpFloor(a, b, x);
    }

    function b3(uint256 x) external pure returns (uint256) {
        return CollRebalancerMathCopy._bezier3(_c(), x);
    }

    function b4(uint256 x) external pure returns (uint256) {
        return CollRebalancerMathCopy._bezier4(_c(), x);
    }

    function m512(uint256 a, uint256 b) external pure returns (uint256, uint256) {
        return Mul512.mul512(a, b);
    }

    function pgt(uint256 a, uint256 b, uint256 c, uint256 d) external pure returns (bool) {
        return Mul512.productGt(a, b, c, d);
    }
}
