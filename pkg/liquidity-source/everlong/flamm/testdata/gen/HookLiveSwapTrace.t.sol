// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @notice Replays the one real Swap on the Base FLAMM pool with `-vvvv` so the trace shows the exact
///         SwapContext core passed to the hook (physicalPoolAsset 219423, maxAmountOut 101043362e12); the
///         context in HookFillGrid.test_liveSwap is transcribed from it.
contract KyberHookLiveSwapTrace is Test {
    bytes32 constant TX = 0x46c3cd72a5860b2fe546e5a2130e066314e3777027151661e1e4f19a935901fa;

    function test_trace() public {
        vm.createSelectFork("https://mainnet.base.org", TX);
        vm.transact(TX);
    }
}
