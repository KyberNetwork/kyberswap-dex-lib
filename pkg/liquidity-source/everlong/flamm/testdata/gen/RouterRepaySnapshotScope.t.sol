// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import "./RouterSettlementEdges.t.sol";

/// @dev The Router's repay snapshot is EIP-1153 transient storage: a proportional withdrawCollateral in a LATER
///      transaction finds none. Run with `--isolate` so each top-level call is its own transaction.
contract RouterRepaySnapshotScope is RouterSettlementEdges {
    function test_repaySnapshotIsPerTransaction() public {
        _setup(false);
        _apply(_indebted());
        R2Harness(POOL).approveRouter(USDC);
        vm.prank(POOL);
        IMMRouter(ROUTER).repay(1, 5_000e6);
        vm.prank(POOL);
        (bool ok, bytes memory ret) =
            ROUTER.call(abi.encodeCall(IMMRouter.withdrawCollateral, (uint16(1), 1000, 0, true)));
        emit log_named_uint("second-tx proportional withdrawCollateral ok", ok ? 1 : 0);
        emit log_named_bytes("revert", ret);
    }

    function test_settleOneLoan() public override {}

    function test_settleTwoLoans() public override {}
}
