// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @notice Shared row writer for the swap hook edge fixtures: one JSON object per call,
///         {"f": fn, "a": [args], "ok": bool, "r": [static uint256 return words], "e": revert data}.
abstract contract SwapHookEdgesBase is Test {
    string internal _out;
    bool internal _firstRow = true;
    uint256 internal _rows;

    function _open(string memory path) internal {
        _out = path;
        _firstRow = true;
        _rows = 0;
        if (vm.exists(path)) vm.removeFile(path);
    }

    function _close() internal {
        vm.writeLine(_out, "]");
    }

    function _q(uint256 v) internal pure returns (string memory) {
        return string.concat('"', vm.toString(v), '"');
    }

    function _arr(uint256[] memory xs) internal pure returns (string memory s) {
        s = "[";
        for (uint256 i; i < xs.length; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", _q(xs[i]));
        }
        s = string.concat(s, "]");
    }

    function _words(bytes memory ret, uint256 n) internal pure returns (uint256[] memory w) {
        w = new uint256[](n);
        for (uint256 i; i < n; ++i) {
            uint256 v;
            assembly ("memory-safe") {
                v := mload(add(ret, add(32, mul(32, i))))
            }
            w[i] = v;
        }
    }

    function _emit(string memory f, uint256[] memory args, bool ok, bytes memory ret, uint256 nOut) internal {
        string memory row = string.concat(
            '{"f":"',
            f,
            '","a":',
            _arr(args),
            ',"ok":',
            ok ? "true" : "false",
            ',"r":',
            ok ? _arr(_words(ret, nOut)) : "[]",
            ',"e":"',
            ok ? "0x" : vm.toString(ret),
            '"}'
        );
        vm.writeLine(_out, string.concat(_firstRow ? "[" : ",", row));
        _firstRow = false;
        ++_rows;
    }

    function _emitRaw(string memory f, uint256[] memory args, bool ok, uint256[] memory outs, bytes memory err)
        internal
    {
        string memory row = string.concat(
            '{"f":"',
            f,
            '","a":',
            _arr(args),
            ',"ok":',
            ok ? "true" : "false",
            ',"r":',
            ok ? _arr(outs) : "[]",
            ',"e":"',
            ok ? "0x" : vm.toString(err),
            '"}'
        );
        vm.writeLine(_out, string.concat(_firstRow ? "[" : ",", row));
        _firstRow = false;
        ++_rows;
    }

    function _rand(uint256 seed, uint256 i) internal pure returns (uint256) {
        return uint256(keccak256(abi.encode(seed, i)));
    }

    /// @dev A log-uniform-ish magnitude in [0, 10**maxExp).
    function _logRand(uint256 r, uint256 maxExp) internal pure returns (uint256) {
        uint256 e = (r >> 200) % (maxExp + 1);
        if (e == 0) return r % 2;
        return r % (10 ** e);
    }

    function _a1(uint256 a) internal pure returns (uint256[] memory x) {
        x = new uint256[](1);
        x[0] = a;
    }

    function _a2(uint256 a, uint256 b) internal pure returns (uint256[] memory x) {
        x = new uint256[](2);
        (x[0], x[1]) = (a, b);
    }

    function _a3(uint256 a, uint256 b, uint256 c) internal pure returns (uint256[] memory x) {
        x = new uint256[](3);
        (x[0], x[1], x[2]) = (a, b, c);
    }
}
