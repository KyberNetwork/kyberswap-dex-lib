// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./FinancingSequenceBase.sol";

interface R3MorphoAccrue {
    function accrueInterest(R3Params memory) external;
}

/// @dev Pseudo-random grid over written market states of the live c104 market (Base fork): the deployed
///      AdaptiveCurveIrm (borrowRateView, and borrowRate as Morpho with the stored endRateAtTarget read back),
///      Morpho Blue's accrueInterest (market and rateAtTarget after), and the deployed MorphoBlueAccount's views
///      (tryPosition, debtOf, suppliedOf, supplySharesToAssets, freeLiquidity, borrowRateAfter). Utilisation spans
///      [0, 1] plus over-full view states, elapsed spans the IRM grace boundary +-1 and multi-year gaps,
///      rateAtTarget spans zero, the MIN / MAX bounds +-1 and arbitrary values, fees and IRM outages are random.
contract MorphoAccrualGrid is FinancingSequenceBase {
    uint256 internal constant ROWS = 900;
    address internal constant ACCOUNT = 0x6760E3b032eE2d670Cb684d9076b8f48cb066c48;
    uint256 internal constant MIN_RAT = uint256(0.001e18) / 365 days;
    uint256 internal constant MAX_RAT = uint256(2e18) / 365 days;

    function _mSlot() internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(LIVE_ID, uint256(3))));
    }

    function _pSlot() internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(ACCOUNT, keccak256(abi.encode(LIVE_ID, uint256(2))))));
    }

    function test_grid() public {
        vm.createSelectFork(RPC, BLOCK);
        seed = 0x6121D;
        open("mm_accrual_grid.jsonl");
        R3Params memory p = params(LIVE_ID);
        for (uint256 i; i < ROWS; ++i) {
            uint256 snap = vm.snapshotState();
            this.row(i, p);
            vm.revertToState(snap);
        }
    }

    function row(uint256 i, R3Params memory p) external {
        uint256 k = (i + 1) * 100;
        uint256 now_ = block.timestamp;
        uint128 tsa = uint128(rnd(k + 1) % 5 == 0 ? rlog(k + 2, 0, type(uint128).max) : rlog(k + 2, 0, 1e16));
        uint256 ur = rnd(k + 3) % 10;
        uint128 tba = ur == 0 ? 0 : ur == 1 ? tsa : ur == 2 ? uint128(uint256(tsa) * 9 / 10) : uint128(rnd(k + 4) % (uint256(tsa) + 1));
        uint128 tss = uint128(rlog(k + 5, 0, uint256(tsa) * 1e6 + 1e6 > type(uint128).max ? type(uint128).max : uint256(tsa) * 1e6 + 1e6));
        uint128 tbs = uint128(rlog(k + 6, 0, uint256(tba) * 1e6 + 1e6 > type(uint128).max ? type(uint128).max : uint256(tba) * 1e6 + 1e6));
        uint256 er = rnd(k + 7) % 10;
        uint256 elapsed = er == 0 ? 0 : er == 1 ? 3599 + rnd(k + 8) % 3 : er == 2 ? 1 : er < 6 ? rlog(k + 8, 1, 4000) : rlog(k + 8, 1, 3e8);
        uint128 fee = rnd(k + 9) % 3 == 0 ? uint128(rlog(k + 10, 1, 0.25e18)) : 0;
        uint256 rr = rnd(k + 11) % 9;
        uint256 rat = rr == 0 ? 0 : rr == 1 ? MIN_RAT : rr == 2 ? MIN_RAT + 1 : rr == 3 ? MAX_RAT : rr == 4 ? MAX_RAT - 1 : rlog(k + 12, 1, 2 * MAX_RAT);
        bool down = rnd(k + 13) % 8 == 0;
        uint256 supShares = rlog(k + 14, 0, tss);
        uint128 borShares = uint128(rlog(k + 15, 0, tbs));
        uint128 coll = uint128(rlog(k + 16, 0, 1e12));

        uint256 m = _mSlot();
        vm.store(MORPHO, bytes32(m), bytes32((uint256(tss) << 128) | tsa));
        vm.store(MORPHO, bytes32(m + 1), bytes32((uint256(tbs) << 128) | tba));
        vm.store(MORPHO, bytes32(m + 2), bytes32((uint256(fee) << 128) | (now_ - elapsed)));
        uint256 ps = _pSlot();
        vm.store(MORPHO, bytes32(ps), bytes32(supShares));
        vm.store(MORPHO, bytes32(ps + 1), bytes32((uint256(coll) << 128) | borShares));
        vm.store(IRM, keccak256(abi.encode(LIVE_ID, uint256(0))), bytes32(rat));
        if (down) {
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRateView.selector, p), "irm down");
            vm.mockCallRevert(IRM, abi.encodeWithSelector(R3Irm.borrowRate.selector, p), "irm down");
        }
        R3Market memory mk = mkt(LIVE_ID);
        uint256 dB = rnd(k + 17) % 4 == 0 ? 0 : rlog(k + 18, 0, rnd(k + 19) % 5 == 0 ? type(uint128).max : uint256(tsa) + 1);
        uint256 dS = rnd(k + 20) % 3 == 0 ? 0 : rnd(k + 21) % 3 == 0 ? tsa : rlog(k + 22, 0, tsa);
        uint256 shares = rlog(k + 23, 0, tss);
        string memory s = string.concat('{"i":', vm.toString(i), ',"t":', q(now_), ',"down":', qb(down), ',"state":', morphoJson(LIVE_ID, ACCOUNT));
        s = string.concat(s, ',"view":', sv(IRM, abi.encodeCall(R3Irm.borrowRateView, (p, mk))));
        s = string.concat(s, ',"acct":{"tryPosition":', sv(ACCOUNT, abi.encodeCall(R3Account.tryPosition, (LIVE_ID))));
        s = string.concat(s, ',"debtOf":', sv(ACCOUNT, abi.encodeCall(R3Account.debtOf, (LIVE_ID))), ',"suppliedOf":', sv(ACCOUNT, abi.encodeCall(R3Account.suppliedOf, (LIVE_ID))));
        s = string.concat(s, ',"shares":', q(shares), ',"s2a":', sv(ACCOUNT, abi.encodeCall(R3Account.supplySharesToAssets, (LIVE_ID, shares))));
        s = string.concat(s, ',"dB":', q(dB), ',"dS":', q(dS), ',"rate":', sv(ACCOUNT, abi.encodeCall(R3Account.borrowRateAfter, (LIVE_ID, dB, dS))), "}");
        uint256 snap = vm.snapshotState();
        vm.prank(MORPHO);
        (bool okr, bytes memory retr) = IRM.call(abi.encodeCall(R3Irm.borrowRate, (p, mk)));
        s = string.concat(s, ',"rate":', res(okr, retr), ',"ratAfter":', q(uint256(R3Irm(IRM).rateAtTarget(LIVE_ID))));
        vm.revertToState(snap);
        (bool oka, bytes memory reta) = MORPHO.call(abi.encodeCall(R3MorphoAccrue.accrueInterest, (p)));
        s = string.concat(s, ',"accrue":', res(oka, reta), ',"after":', morphoJson(LIVE_ID, ACCOUNT), "}");
        vm.clearMockedCalls();
        line(s);
    }
}
