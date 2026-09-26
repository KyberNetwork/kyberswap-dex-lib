// SPDX-License-Identifier: UNLICENSED
// Copyright (c) 2025-2026 Everlong Labs Limited
pragma solidity ^0.8.26;

import {Test} from "forge-std/Test.sol";

/// @notice Row writer shared by the leverage edge generators. A fixture is a JSON object whose `rows`
///         array holds `[op, [inputs...], [status, outputs...]]`. status 0 carries every ABI return word of the
///         call verbatim; status 1 carries the revert as [len, selector, first argument word].
abstract contract LevEdgeRows is Test {
    string internal outPath;
    uint256 internal rowCount;
    uint256 internal seed;

    function _begin(string memory path, string memory header) internal {
        outPath = path;
        if (vm.exists(path)) vm.removeFile(path);
        vm.writeLine(path, string.concat("{", header, ',"rows":['));
    }

    function _end() internal {
        vm.writeLine(outPath, string.concat('["end",[],[]]],"count":', vm.toString(rowCount), "}"));
    }

    function _join(uint256[] memory xs) internal pure returns (string memory s) {
        for (uint256 i; i < xs.length; ++i) {
            s = string.concat(s, i == 0 ? "" : ",", vm.toString(xs[i]));
        }
    }

    function _writeRow(string memory op, uint256[] memory ins, uint256[] memory outs) internal {
        uint256 fmp;
        assembly {
            fmp := mload(0x40)
        }
        vm.writeLine(outPath, string.concat('["', op, '",[', _join(ins), "],[", _join(outs), "]],"));
        ++rowCount;
        // The row string is a temporary: hand its memory back so long generators stay under the memory limit.
        assembly {
            mstore(0x40, fmp)
        }
    }

    /// @dev Executes `data` against `target` as a static call and records it. Returns the success flag and data.
    function _call(string memory op, address target, bytes memory data, uint256[] memory ins)
        internal
        returns (bool ok, bytes memory ret)
    {
        (ok, ret) = target.staticcall(data);
        _writeRow(op, ins, _outs(ok, ret));
    }

    function _outs(bool ok, bytes memory ret) internal pure returns (uint256[] memory outs) {
        if (ok) {
            uint256 n = ret.length / 32;
            outs = new uint256[](n + 1);
            for (uint256 i; i < n; ++i) {
                outs[i + 1] = _word(ret, i * 32);
            }
            return outs;
        }
        outs = new uint256[](4);
        outs[0] = 1;
        outs[1] = ret.length;
        if (ret.length >= 4) outs[2] = _word(ret, 0) >> 224;
        if (ret.length >= 36) outs[3] = _word(ret, 4);
    }

    function _word(bytes memory b, uint256 off) internal pure returns (uint256 w) {
        if (b.length < off + 32) {
            // Left-aligned partial word (only used for short revert data).
            for (uint256 i = off; i < b.length; ++i) {
                w |= uint256(uint8(b[i])) << (8 * (31 - (i - off)));
            }
            return w;
        }
        assembly {
            w := mload(add(add(b, 32), off))
        }
    }

    function _rand() internal returns (uint256) {
        // The salt string is part of the recorded seed stream: changing it changes the fixtures.
        seed = uint256(keccak256(abi.encode(seed, "verify_m3_r1")));
        return seed;
    }

    function _a(uint256 a) internal pure returns (uint256[] memory r) {
        r = new uint256[](1);
        r[0] = a;
    }

    function _a(uint256 a, uint256 b) internal pure returns (uint256[] memory r) {
        r = new uint256[](2);
        (r[0], r[1]) = (a, b);
    }

    function _a(uint256 a, uint256 b, uint256 c) internal pure returns (uint256[] memory r) {
        r = new uint256[](3);
        (r[0], r[1], r[2]) = (a, b, c);
    }

    function _a(uint256 a, uint256 b, uint256 c, uint256 d) internal pure returns (uint256[] memory r) {
        r = new uint256[](4);
        (r[0], r[1], r[2], r[3]) = (a, b, c, d);
    }

    function _a(uint256 a, uint256 b, uint256 c, uint256 d, uint256 e) internal pure returns (uint256[] memory r) {
        r = new uint256[](5);
        (r[0], r[1], r[2], r[3], r[4]) = (a, b, c, d, e);
    }

    function _a(uint256 a, uint256 b, uint256 c, uint256 d, uint256 e, uint256 f)
        internal
        pure
        returns (uint256[] memory r)
    {
        r = new uint256[](6);
        (r[0], r[1], r[2], r[3], r[4], r[5]) = (a, b, c, d, e, f);
    }

    function _a7(uint256[7] memory x) internal pure returns (uint256[] memory r) {
        r = new uint256[](7);
        for (uint256 i; i < 7; ++i) {
            r[i] = x[i];
        }
    }
}
