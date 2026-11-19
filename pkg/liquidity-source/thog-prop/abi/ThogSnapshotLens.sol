// SPDX-License-Identifier: MIT
pragma solidity 0.8.25;

interface IThogSnapshot {
    struct TokenMetadata {
        address token;
        uint8 decimals;
        uint8 category;
        uint32 price;
        uint32 priceTail;
        uint32 tokenBidSpread;
        uint32 tokenAskSpread;
        uint32 volumeRate;
        uint32 categorySpread;
        uint32 inventory;
        uint32 sides;
        uint32 postedBlock;
        uint32 sequence;
        uint32 pairStart;
        uint32 inventoryCap;
    }
    struct CategoryMetadata {
        uint8 anchorTokenIndex;
        uint32 price;
        uint32 priceTail;
        uint32 inventory;
        uint32 spread;
        uint32 postedBlock;
        uint32 inventoryCap;
    }
    struct StorageWord { uint16 slot; uint256 value; }
    struct PoolData {
        uint16 schemaVersion;
        uint256 blockNumber;
        uint256 baseFeePerGas;
        bytes32 parentBlockHash;
        uint8 tokenCount;
        uint8 categoryCount;
        uint8 startTokenIndex;
        TokenMetadata[] tokens;
        uint256[] balances;
        CategoryMetadata[] categories;
        StorageWord[] words;
        uint256 nativeBalance;
        uint256[] portfolioBalances;
    }
    function getPoolData(uint8 start, uint8 count) external view returns (PoolData memory);
    function storedMultiplier(address token) external view returns (uint256);
}

/// @notice Never deployed: run as the init code of one creation eth_call, so every read sees one EVM state.
/// @dev solc 0.8.25, shanghai, viaIR, optimizer 200, metadata bytecodeHash none.
/// A successful read reverts with abi.encode(bytes4("THG8"), block.number, block.timestamp,
/// rawGetPoolDataResult, shareTokens, shares, multipliers). All failure payloads are distinct.
contract ThogSnapshotLens {
    error ReadFailed(address target, bytes4 selector, bytes reason);
    error InvalidPage();

    constructor(address proxy) payable {
        bytes memory page = read(proxy, abi.encodeCall(IThogSnapshot.getPoolData, (0, 28)));
        IThogSnapshot.PoolData memory data = abi.decode(page, (IThogSnapshot.PoolData));
        if (data.schemaVersion != 8 || data.blockNumber != block.number || data.baseFeePerGas != block.basefee
            || data.startTokenIndex != 0 || data.tokens.length != data.tokenCount || data.tokenCount > 28) {
            revert InvalidPage();
        }
        uint256 listed = data.tokens.length;
        address[] memory shareTokens = new address[](listed);
        uint256[] memory shares = new uint256[](listed);
        uint256[] memory multipliers = new uint256[](listed);
        uint256 count;
        for (uint256 i; i < listed; ++i) {
            address token = data.tokens[i].token;
            // The pool stores a multiplier only for tokens it holds as shares; zero is an ordinary ERC-20.
            uint256 multiplier = abi.decode(read(proxy, abi.encodeCall(IThogSnapshot.storedMultiplier, (token))), (uint256));
            if (multiplier == 0) continue;
            shareTokens[count] = token;
            shares[count] = abi.decode(read(token, abi.encodeWithSignature("sharesOf(address)", proxy)), (uint256));
            multipliers[count] = multiplier;
            ++count;
        }
        assembly ("memory-safe") {
            mstore(shareTokens, count)
            mstore(shares, count)
            mstore(multipliers, count)
        }
        bytes memory result = abi.encode(bytes4("THG8"), block.number, block.timestamp, page, shareTokens, shares, multipliers);
        assembly ("memory-safe") { revert(add(result, 32), mload(result)) }
    }

    function read(address target, bytes memory input) private view returns (bytes memory output) {
        bool success;
        (success, output) = target.staticcall(input);
        if (!success) revert ReadFailed(target, bytes4(input), output);
    }
}
