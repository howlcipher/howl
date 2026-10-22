#!/usr/bin/env python3
"""Exercise real Howl -> installed HowlPlane process behavior in isolated state."""
import argparse
import json
import os
from pathlib import Path
import subprocess
import tempfile


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--howl", required=True)
    args = parser.parse_args()
    binary = str(Path(args.howl).resolve())
    with tempfile.TemporaryDirectory(prefix="howl-interop-runtime-") as temporary:
        root = Path(temporary)
        repo = root / "repo"
        repo.mkdir()
        env = dict(os.environ)
        # Every writable user/state path belongs to this disposable environment.
        for name, directory in {"HOME": "home", "XDG_STATE_HOME": "state",
                                "XDG_DATA_HOME": "data", "XDG_CONFIG_HOME": "config",
                                "XDG_CACHE_HOME": "cache"}.items():
            env[name] = str(root / directory)
        subprocess.run(["git", "init", "--quiet", str(repo)], env=env, check=True)
        def snapshot():
            return {str(p.relative_to(repo)): p.read_bytes()
                    for p in repo.rglob("*") if p.is_file()}
        before = snapshot()
        def invoke(operation):
            return subprocess.run([binary, "orchestrate", operation, "--repo", str(repo), "--json"],
                                  env=env, text=True, capture_output=True, timeout=30)
        for _ in range(2):
            inspected = invoke("inspect")
            assert inspected.returncode == 0, "real inspection failed"
            assert json.loads(inspected.stdout) == [], "unexpected pre-existing sessions"
            failed = invoke("resume")
            assert failed.returncode == 1, f"missing session returned {failed.returncode}"
            assert "No unfinished session" in failed.stdout + failed.stderr
        assert snapshot() == before, "read-only forwarding changed repository state"
        print(json.dumps({"real_components": ["howl", "howlplane"],
                          "inspection_runs": 2, "expected_failures": 2,
                          "repository_unchanged": True}))


if __name__ == "__main__":
    main()
