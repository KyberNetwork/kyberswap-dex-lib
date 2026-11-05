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
}

/// @notice Never deployed. Read the page and its listed xStock issuers in one EVM state.
/// @dev solc 0.8.25, shanghai, viaIR, optimizer 200, metadata bytecodeHash none.
/// Successful reads revert with abi.encode(bytes4("THG8"), block.number, block.timestamp,
/// rawGetPoolDataResult, issuers, shares, multipliers). All failure payloads are distinct.
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
        uint256 count;
        for (uint256 i; i < data.tokens.length; ++i) {
            if (isXStock(data.tokens[i].token)) ++count;
        }
        address[] memory issuers = new address[](count);
        uint256[] memory shares = new uint256[](count);
        uint256[] memory multipliers = new uint256[](count);
        uint256 cursor;
        for (uint256 i; i < data.tokens.length; ++i) {
            address token = data.tokens[i].token;
            if (!isXStock(token)) continue;
            issuers[cursor] = token;
            shares[cursor] = abi.decode(read(token, abi.encodeWithSignature("sharesOf(address)", proxy)), (uint256));
            (multipliers[cursor],,) = abi.decode(read(token, abi.encodeWithSignature("getCurrentMultiplier()")), (uint256, uint256, uint256));
            ++cursor;
        }
        bytes memory result = abi.encode(bytes4("THG8"), block.number, block.timestamp, page, issuers, shares, multipliers);
        assembly ("memory-safe") { revert(add(result, 32), mload(result)) }
    }

    function read(address target, bytes memory input) private view returns (bytes memory output) {
        bool success;
        (success, output) = target.staticcall(input);
        if (!success) revert ReadFailed(target, bytes4(input), output);
    }

    // The deployed catalog identifies issuer tokens; the page selects active listings.
    function isXStock(address token) private pure returns (bool) {
        return token == address(bytes20(hex"90a2a4c76b5d8c0bc892a69ea28aa775a8f2dd48"))
            || token == address(bytes20(hex"a753a7395cae905cd615da0b82a53e0560f250af"))
            || token == address(bytes20(hex"8d512a03600981b8e86556fd5c0fbdd726fe1821"))
            || token == address(bytes20(hex"299c95eb52bfd5bc98d924bce17197b64864589b"))
            || token == address(bytes20(hex"c845b2894dbddd03858fd2d643b4ef725fe0849d"))
            || token == address(bytes20(hex"8ad3c73f833d3f9a523ab01476625f269aeb7cf0"))
            || token == address(bytes20(hex"68fa48b1c2fe52b3d776e1953e0e782b5044ce28"))
            || token == address(bytes20(hex"f6a873bae4ba1b304e45df52a4b7d176e1c6a8c4"))
            || token == address(bytes20(hex"58100046a4afcd4ee4fadbd4244f3f895a341c56"));
    }
}
