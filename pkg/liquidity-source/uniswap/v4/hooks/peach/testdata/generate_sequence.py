#!/usr/bin/env python3
"""Generate real PoolManager swap references without changing the source checkout."""

import argparse
import pathlib
import shutil
import subprocess
import tempfile

SOURCE_COMMIT = "314ed2151367d3edb6f9045f05f5eb18567f5aa6"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--source", required=True, type=pathlib.Path)
    args = parser.parse_args()
    source = args.source.resolve()
    here = pathlib.Path(__file__).resolve().parent
    commit = subprocess.check_output(
        ["git", "-C", str(source), "rev-parse", "HEAD"], text=True
    ).strip()
    if commit != SOURCE_COMMIT:
        raise SystemExit(f"Expected pinned public source commit {SOURCE_COMMIT}")
    if subprocess.check_output(
        ["git", "-C", str(source), "status", "--porcelain", "--untracked-files=no"],
        text=True,
    ).strip():
        raise SystemExit("Expected an unmodified source checkout and dependencies")

    with tempfile.TemporaryDirectory(prefix="peach-sequence-") as directory:
        root = pathlib.Path(directory)
        (root / "test").mkdir()
        shutil.copyfile(here / "GenerateSequence.t.sol.txt", root / "test/GenerateSequence.t.sol")
        config = (source / "foundry.toml").read_text()
        config = config.replace('src = "src"', 'src = "test"')
        config = config.replace('path = "test/fixtures"', 'path = "."')
        config = config.replace('access = "read"', 'access = "read-write"')
        config = config.replace('remappings = [', f'remappings = [\n    "peach-source/={source}/",')
        config = config.replace("=lib/", f"={source}/lib/")
        (root / "foundry.toml").write_text(config)
        subprocess.run(
            ["forge", "test", "--root", str(root), "--match-contract", "GenerateSequence",
             "--match-test", "test_generateKyberSequence", "-vv"],
            check=True,
        )
        shutil.copyfile(root / "sequence.json", here / "sequence.json")


if __name__ == "__main__":
    main()
