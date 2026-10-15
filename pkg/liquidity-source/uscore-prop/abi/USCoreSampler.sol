// SPDX-License-Identifier: MIT
pragma solidity 0.8.30;

interface IUSCorePool {
    function getReserves() external view returns (uint256 reserveQuote, uint256 reserveBase);

    function quoteExactIn(address tokenIn, uint256 amountIn)
        external
        view
        returns (uint256 amountOut, uint256 fee, uint8 status);

    function quoteLadder(address tokenIn, uint256[] calldata amountsIn)
        external
        view
        returns (uint256[] memory outs, uint256[] memory fees, uint8 status);
}

/// @title USCoreSampler
/// @notice A deployless helper that samples a USCore pool's quote curve in one eth_call. Its creation code runs as a
/// to-less eth_call and reverts with Sample, never deployed.
/// @dev Mirrors the tracker's adaptive sampling, all at one block: a 1-token probe per direction (halving down when
/// the pool can't pay that much) sets a basis, a geometric grid of bps of it seeds the samples, then up to maxRounds
/// rounds of quoteLadder bisect the capacity edge and add samples where the curve bends most. Sample holds the last
/// round. A pool call that reverts, or a ladder whose lengths don't match, reverts without data.
contract USCoreSampler {
    struct Snapshot {
        uint256 blockNumber;
        uint256 timestamp;
        uint256 reserveQuote;
        uint256 reserveBase;
        uint256[] points0;
        uint256[] outs0;
        uint8 status0;
        uint256[] points1;
        uint256[] outs1;
        uint8 status1;
        uint8 rounds;
    }

    error Sample(Snapshot snapshot);

    uint8 internal constant OK = 0;
    uint8 internal constant INSUFFICIENT = 3; // the pool can't pay out this amount
    uint8 internal constant MAX_STATUS = 5;
    uint256 internal constant MAX_POINT = type(uint128).max;
    uint256 internal constant ONE = 1 << 64; // fixed-point 1.0 of the curvature score
    uint256 internal constant SCORE_THRESHOLD = ONE / 100000;
    uint256 internal constant SCORE_CAP = 1 << 190;
    uint256 internal constant UPPER_SPLITS = 8;

    constructor(
        IUSCorePool pool,
        address[2] memory tokens,
        uint256[2] memory units,
        uint256[] memory bps,
        uint256 maxPoints,
        uint256 maxRounds
    ) {
        uint256[][2] memory points = initialPoints(pool, tokens, units, bps);
        uint256[][2] memory outs;
        uint8[2] memory status;
        Snapshot memory s;
        for (uint256 round = 0;; round++) {
            (s.reserveQuote, s.reserveBase) = pool.getReserves();
            for (uint256 i = 0; i < 2; i++) {
                (outs[i], status[i]) = quoteLadder(pool, tokens[i], points[i]);
            }
            s.rounds = uint8(round + 1);
            if (round == maxRounds || status[0] > MAX_STATUS || status[1] > MAX_STATUS) break;
            bool changed;
            for (uint256 i = 0; i < 2; i++) {
                // Locate the capacity edge first; from the third round on, only reshape the curve.
                uint256[] memory refined = refineUpper(points[i], outs[i], status[i], maxPoints);
                if (round == 2 || refined.length <= points[i].length) {
                    refined = refineCurve(points[i], outs[i], status[i], maxPoints);
                }
                if (refined.length > 0 && !equal(refined, points[i])) {
                    points[i] = refined;
                    changed = true;
                }
            }
            if (!changed) break;
        }
        s.blockNumber = block.number;
        s.timestamp = block.timestamp;
        (s.points0, s.outs0, s.status0) = (points[0], outs[0], status[0]);
        (s.points1, s.outs1, s.status1) = (points[1], outs[1], status[1]);
        revert Sample(s);
    }

    function quoteLadder(IUSCorePool pool, address token, uint256[] memory points)
        internal
        view
        returns (uint256[] memory outs, uint8 status)
    {
        if (points.length == 0) return (outs, OK);
        uint256[] memory fees;
        (outs, fees, status) = pool.quoteLadder(token, points);
        if (outs.length != points.length || fees.length != points.length) revert();
    }

    /// @dev Spreads bps of the input that would drain the output reserve at the 1-token rate.
    function initialPoints(IUSCorePool pool, address[2] memory tokens, uint256[2] memory units, uint256[] memory bps)
        internal
        view
        returns (uint256[][2] memory points)
    {
        (uint256 r0, uint256 r1) = pool.getReserves();
        uint256[2] memory reserves = [r0, r1];
        for (uint256 i = 0; i < 2; i++) {
            (uint256 out,, uint8 status) = pool.quoteExactIn(tokens[i], units[i]);
            uint256 basis = reserves[i];
            uint256 minimum;
            if (reserves[1 - i] != 0 && status == INSUFFICIENT) {
                (uint256 rateIn, uint256 rateOut, uint256 low) = smallerProbe(pool, tokens[i], units[i]);
                if (rateOut != 0) {
                    minimum = low;
                    basis = mulDivCapped(reserves[1 - i], rateIn, rateOut);
                }
            }
            if (status == OK && out > 0) basis = mulDivCapped(reserves[1 - i], units[i], out);
            if (basis > MAX_POINT) basis = MAX_POINT;

            uint256[] memory grid = new uint256[](bps.length + 1);
            uint256 n;
            if (minimum != 0) grid[n++] = minimum;
            for (uint256 k = 0; k < bps.length; k++) {
                uint256 x = basis * bps[k] / 10000;
                if (x > 0 && x > minimum && (n == 0 || x != grid[n - 1])) grid[n++] = x;
            }
            points[i] = shrink(grid, n);
        }
    }

    /// @dev Quotes unit/2, unit/4, ... for pools too shallow to quote one token: the largest quotable amount gives
    /// the rate, the smallest is kept as the first sample.
    function smallerProbe(IUSCorePool pool, address token, uint256 unit)
        internal
        view
        returns (uint256 rateIn, uint256 rateOut, uint256 minimum)
    {
        for (uint256 amount = unit >> 1; amount > 0; amount >>= 1) {
            (uint256 out,, uint8 status) = pool.quoteExactIn(token, amount);
            if (status == OK && out > 0) {
                if (rateOut == 0) (rateIn, rateOut) = (amount, out);
                minimum = amount;
            }
        }
    }

    /// @dev Bisects the gap where a partial (INSUFFICIENT) ladder first fails, in eighths.
    function refineUpper(uint256[] memory points, uint256[] memory outs, uint8 status, uint256 maxPoints)
        internal
        pure
        returns (uint256[] memory refined)
    {
        if (status != INSUFFICIENT || points.length >= maxPoints) return refined;
        for (uint256 i = 0; i < outs.length; i++) {
            if (outs[i] != 0) {
                if (i > 0 && outs[i] < outs[i - 1]) return refined;
                continue;
            }
            if (i == 0) return refined;
            refined = new uint256[](maxPoints);
            uint256 n;
            for (; n < i; n++) {
                refined[n] = points[n];
            }
            (uint256 lo, uint256 hi) = (points[i - 1], points[i]);
            for (uint256 k = 1; k < UPPER_SPLITS && n + points.length - i < maxPoints; k++) {
                uint256 x = (hi - lo) * k / UPPER_SPLITS + lo;
                if (x > refined[n - 1] && x < hi) refined[n++] = x;
            }
            for (uint256 j = i; j < points.length; j++) {
                refined[n++] = points[j];
            }
            return shrink(refined, n);
        }
    }

    /// @dev Spends the sample budget on segments whose rate differs most from their neighbours', after dropping
    /// samples past the first zero or decreasing output (they never reach the ladder). A segment's score is
    /// |rate_i - rate_j| * width_i / out_i = |delta_i - delta_j * width_i / width_j| / out_i, in 64-bit fixed point;
    /// each split halves the segment and quarters its score.
    function refineCurve(uint256[] memory points, uint256[] memory outs, uint8 status, uint256 maxPoints)
        internal
        pure
        returns (uint256[] memory refined)
    {
        if (points.length == 0 || (status != OK && status != INSUFFICIENT)) return refined;
        uint256 n;
        uint256[] memory lo = new uint256[](outs.length);
        uint256[] memory width = new uint256[](outs.length);
        int256[] memory delta = new int256[](outs.length);
        for (uint256 i = 0; i < outs.length; i++) {
            if (outs[i] == 0 || outs[i] > MAX_POINT) break;
            (uint256 l, uint256 prev) = i > 0 ? (points[i - 1], outs[i - 1]) : (0, 0);
            (lo[n], width[n], delta[n]) = (l, points[i] - l, int256(outs[i]) - int256(prev));
            n++;
            if (delta[n - 1] < 0) break;
        }
        uint256 kept = n > 0 && delta[n - 1] < 0 ? n : n + 1;
        if (kept > points.length) kept = points.length;

        uint256[] memory score = new uint256[](n);
        for (uint256 i = 1; i < n; i++) {
            if (width[i] < 2) continue;
            for (uint256 j = i - 1; j <= i + 1 && j < n; j += 2) {
                uint256 mag = abs(delta[j]) * width[i] / width[j];
                if (mag >= SCORE_CAP) {
                    score[i] = SCORE_CAP;
                    break;
                }
                int256 scaled = delta[j] < 0 ? -int256(mag) : int256(mag);
                uint256 sc = abs(delta[i] - scaled) * ONE / outs[i];
                if (sc > SCORE_CAP) sc = SCORE_CAP;
                if (sc > score[i]) score[i] = sc;
            }
        }

        uint256[] memory parts = new uint256[](n);
        for (uint256 i = 0; i < n; i++) {
            parts[i] = 1;
        }
        for (uint256 budget = maxPoints - kept; budget > 0;) {
            uint256 best = type(uint256).max;
            for (uint256 i = 0; i < n; i++) {
                if (
                    parts[i] <= budget && score[i] > SCORE_THRESHOLD
                        && (best == type(uint256).max || score[i] > score[best])
                ) best = i;
            }
            if (best == type(uint256).max) break;
            budget -= parts[best];
            parts[best] *= 2;
            score[best] >>= 2;
        }

        // Segment i ends at points[i], so its new samples go right before it.
        refined = new uint256[](maxPoints);
        uint256 m;
        for (uint256 i = 0; i < kept; i++) {
            if (i < n) {
                for (uint256 k = 1; k < parts[i]; k++) {
                    uint256 x = width[i] * k / parts[i] + lo[i];
                    if (x > lo[i] && x < points[i] && (m == 0 || x != refined[m - 1])) refined[m++] = x;
                }
            }
            if (m == 0 || points[i] != refined[m - 1]) refined[m++] = points[i];
        }
        return shrink(refined, m);
    }

    function equal(uint256[] memory a, uint256[] memory b) internal pure returns (bool) {
        if (a.length != b.length) return false;
        for (uint256 i = 0; i < a.length; i++) {
            if (a[i] != b[i]) return false;
        }
        return true;
    }

    function shrink(uint256[] memory a, uint256 n) internal pure returns (uint256[] memory) {
        assembly {
            mstore(a, n)
        }
        return a;
    }

    function abs(int256 x) internal pure returns (uint256) {
        return x < 0 ? uint256(-x) : uint256(x);
    }

    /// @dev x * y / d rounded down with a 512-bit product, MAX_POINT when the result overflows 256 bits.
    function mulDivCapped(uint256 x, uint256 y, uint256 d) internal pure returns (uint256 result) {
        unchecked {
            uint256 prod0 = x * y;
            uint256 mm = mulmod(x, y, type(uint256).max);
            uint256 prod1 = mm - prod0 - (mm < prod0 ? 1 : 0);
            if (prod1 == 0) return prod0 / d;
            if (prod1 >= d) return MAX_POINT;
            uint256 remainder = mulmod(x, y, d);
            prod1 -= remainder > prod0 ? 1 : 0;
            prod0 -= remainder;
            uint256 twos = d & (~d + 1);
            d /= twos;
            prod0 /= twos;
            twos = (0 - twos) / twos + 1;
            prod0 |= prod1 * twos;
            uint256 inv = (3 * d) ^ 2;
            inv *= 2 - d * inv;
            inv *= 2 - d * inv;
            inv *= 2 - d * inv;
            inv *= 2 - d * inv;
            inv *= 2 - d * inv;
            inv *= 2 - d * inv;
            result = prod0 * inv;
        }
    }
}
