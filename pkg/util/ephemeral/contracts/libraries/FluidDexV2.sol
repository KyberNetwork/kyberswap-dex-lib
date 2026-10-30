// SPDX-License-Identifier: MIT
pragma solidity ^0.8.37;

/// @notice The one accessor a lens reads Fluid DEX v2 through. The DEX keeps every pool's state in
/// its own storage and exposes no getter for it, only this raw read.
interface IFluidDexV2Reads {
    function readFromStorage(bytes32 slot) external view returns (uint256);
}

/// @title FluidDexV2
/// @notice The reads a lens makes of a Fluid DEX v2 pool (a D3 or D4 dex type), through
/// `readFromStorage`, and the storage layout they rely on.
/// @dev The slots and bit positions are Fluid's DexV2D3D4CommonSlotsLink. Every pool's state is a
/// mapping entry keyed by dex type, then dex id: `_dexVariables` (0), `_dexVariables2` (1),
/// `_tickBitmap` (2, then by tick over the spacing >> 8), `_tickLiquidityGross` (3, then by tick),
/// `_tickData` (4, then by tick, liquidityNet first) and `_dexKey` (8). A layout the DEX changed in
/// an upgrade shows as a tick spacing out of range or a set bit without liquidity, which the lens
/// refuses rather than read on.
library FluidDexV2 {
    /// @notice A reply came back shorter than a word, i.e. the address is not the DEX.
    error UnexpectedDexResponse();

    /// @notice The tick bounds, narrower than Uniswap's: a tick is 19 bits and a sign.
    int256 internal constant MIN_TICK = -524287;
    int256 internal constant MAX_TICK = 524287;

    /// @notice The tick spacings a pool may have.
    int24 internal constant MIN_TICK_SPACING = 1;
    int24 internal constant MAX_TICK_SPACING = 500;

    uint256 private constant DEX_VARIABLES_SLOT = 0;
    uint256 private constant DEX_VARIABLES2_SLOT = 1;
    uint256 internal constant TICK_BITMAP_SLOT = 2;
    uint256 internal constant TICK_LIQUIDITY_GROSS_SLOT = 3;
    uint256 internal constant TICK_DATA_SLOT = 4;
    uint256 private constant DEX_KEY_SLOT = 8;

    uint256 private constant X19 = 0x7ffff;
    uint256 private constant X72 = 0xffffffffffffffffff;
    uint256 private constant X102 = 0x3fffffffffffffffffffffffff;

    /// @dev The slot of a pool's entry in the mapping at `slot`: the value itself for a double
    /// mapping, the base of the per-tick keys for a triple one.
    function poolSlot(uint256 slot, uint256 dexType, bytes32 dexId) internal pure returns (bytes32) {
        return keccak256(abi.encode(dexId, keccak256(abi.encode(dexType, slot))));
    }

    /// @dev The slot of `key`'s entry under a pool's base `base` (see poolSlot).
    function keySlot(bytes32 base, int256 key) internal pure returns (bytes32 slot) {
        assembly ("memory-safe") {
            mstore(0, key)
            mstore(0x20, base)
            slot := keccak256(0, 0x40)
        }
    }

    /// @dev The pool's price, tick, in-range liquidity and tick spacing, as the DEX packs them:
    /// `_dexVariables` holds the tick as a sign bit and 19 bits, then the sqrt price as a 64-bit
    /// coefficient and an 8-bit exponent; `_dexVariables2` holds the active liquidity in bits
    /// 38..139; the key's second word holds the tick spacing in bits 184..207.
    function state(address dex, uint256 dexType, bytes32 dexId)
        internal
        view
        returns (uint160 sqrtPriceX96, int24 tick, uint128 liquidity, int24 tickSpacing)
    {
        uint256 variables = read(dex, poolSlot(DEX_VARIABLES_SLOT, dexType, dexId));
        uint256 variables2 = read(dex, poolSlot(DEX_VARIABLES2_SLOT, dexType, dexId));
        uint256 key1 = read(dex, bytes32(uint256(poolSlot(DEX_KEY_SLOT, dexType, dexId)) + 1));

        // forge-lint: disable-start(unsafe-typecast): each value is masked to its field's width,
        // and the widths fit: 19 bits of tick, a price below 2^160, 102 bits of liquidity, 24 of
        // spacing.
        int256 absTick = int256((variables >> 1) & X19);
        tick = int24((variables & 1) == 1 ? absTick : -absTick);
        uint256 price = (variables >> 20) & X72;
        sqrtPriceX96 = uint160((price >> 8) << (price & 0xff));
        liquidity = uint128((variables2 >> 38) & X102);
        tickSpacing = int24(int256((key1 >> 184) & 0xffffff));
        // forge-lint: disable-end(unsafe-typecast)
    }

    /// @dev One word of the pool's tick bitmap, `bitmapBase` being poolSlot(TICK_BITMAP_SLOT, ...).
    function bitmapWord(address dex, bytes32 bitmapBase, int256 wordPos) internal view returns (uint256) {
        return read(dex, keySlot(bitmapBase, wordPos));
    }

    /// @dev A tick's liquidityGross and liquidityNet, the first word of its TickData.
    function tickLiquidity(address dex, bytes32 grossBase, bytes32 dataBase, int256 tick)
        internal
        view
        returns (uint128 liquidityGross, int128 liquidityNet)
    {
        // forge-lint: disable-start(unsafe-typecast): the DEX bounds a tick's liquidity to 102 bits.
        liquidityGross = uint128(read(dex, keySlot(grossBase, tick)));
        liquidityNet = int128(int256(read(dex, keySlot(dataBase, tick))));
        // forge-lint: disable-end(unsafe-typecast)
    }

    /// @dev `dex.readFromStorage(slot)`. A failed call's revert is passed up unchanged.
    function read(address dex, bytes32 slot) internal view returns (uint256 value) {
        bytes4 selector = IFluidDexV2Reads.readFromStorage.selector;
        uint256 size;
        assembly ("memory-safe") {
            // Scratch space only: selector at 0, slot at 4, reply over both.
            mstore(0, selector)
            mstore(4, slot)
            if iszero(staticcall(gas(), dex, 0, 0x24, 0, 0x20)) {
                let ptr := mload(0x40)
                returndatacopy(ptr, 0, returndatasize())
                // The loop is the caller's scan, and a DEX that refuses a read must end it.
                // forge-lint: disable-next-line(require-revert-in-loop)
                revert(ptr, returndatasize())
            }
            size := returndatasize()
            value := mload(0)
        }
        // As above: a short reply ends the scan.
        // forge-lint: disable-next-line(require-revert-in-loop)
        require(size >= 0x20, UnexpectedDexResponse());
    }

    /// @notice Intersects `[wordPosLower, wordPosUpper]` with the words `tickSpacing` can populate
    /// within the DEX's tick bounds, as TickBitmap.clampWordRange does for Uniswap's: the spacing-1
    /// range of a Uniswap pool would read ~2,800 words past Fluid's bounds.
    function clampWordRange(int16 wordPosLower, int16 wordPosUpper, int24 tickSpacing)
        internal
        pure
        returns (int16 lower, int16 upper, bool empty)
    {
        int256 spacing = int256(tickSpacing);
        int256 minCompressed = MIN_TICK / spacing;
        if (MIN_TICK % spacing != 0) --minCompressed;
        int256 minWord = minCompressed >> 8;
        int256 maxWord = (MAX_TICK / spacing) >> 8;

        int256 lo = int256(wordPosLower) > minWord ? int256(wordPosLower) : minWord;
        int256 hi = int256(wordPosUpper) < maxWord ? int256(wordPosUpper) : maxWord;
        if (lo > hi) return (0, 0, true);

        // Both lie between the caller's int16 bounds, so they fit.
        // forge-lint: disable-next-line(unsafe-typecast)
        return (int16(lo), int16(hi), false);
    }
}
