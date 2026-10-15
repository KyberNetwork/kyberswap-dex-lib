// SPDX-License-Identifier: MIT
pragma solidity 0.8.30;

/// @notice Test double of a USCore pool: per direction, out = x*num/den - fee with fee = gross*x/(2*peak), so the
/// output peaks at x = peak and falls after it; above 2*peak, past the reserve, or at zero out it reports 3.
/// pause makes every quote report 4. The tracker tests price the same curve in Go.
contract MockUSCorePool {
    uint256 public immutable reserve0;
    uint256 public immutable reserve1;
    address internal immutable token0;
    uint256 internal immutable num0;
    uint256 internal immutable den0;
    uint256 internal immutable peak0;
    uint256 internal immutable num1;
    uint256 internal immutable den1;
    uint256 internal immutable peak1;
    bool internal immutable paused;

    constructor(address t0, uint256[2] memory reserves, uint256[3] memory c0, uint256[3] memory c1, bool pause) {
        (token0, reserve0, reserve1, paused) = (t0, reserves[0], reserves[1], pause);
        (num0, den0, peak0) = (c0[0], c0[1], c0[2]);
        (num1, den1, peak1) = (c1[0], c1[1], c1[2]);
    }

    function getReserves() external view returns (uint256, uint256) {
        return (reserve0, reserve1);
    }

    function quoteExactIn(address tokenIn, uint256 amountIn) public view returns (uint256 out, uint256 fee, uint8 status) {
        if (paused) return (0, 0, 4);
        bool zero = tokenIn == token0;
        (uint256 num, uint256 den, uint256 peak, uint256 reserveOut) =
            zero ? (num0, den0, peak0, reserve1) : (num1, den1, peak1, reserve0);
        if (amountIn > 2 * peak) return (0, 0, 3);
        uint256 gross = amountIn * num / den;
        fee = gross * amountIn / (2 * peak);
        out = gross - fee;
        if (out == 0 || out > reserveOut) return (0, 0, 3);
    }

    function quoteLadder(address tokenIn, uint256[] calldata amountsIn)
        external
        view
        returns (uint256[] memory outs, uint256[] memory fees, uint8 status)
    {
        outs = new uint256[](amountsIn.length);
        fees = new uint256[](amountsIn.length);
        for (uint256 i = 0; i < amountsIn.length; i++) {
            uint8 s;
            (outs[i], fees[i], s) = quoteExactIn(tokenIn, amountsIn[i]);
            if (s != 0) status = s;
        }
    }
}
