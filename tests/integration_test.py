#!/usr/bin/env python3
"""
Integration tests for artf.

Starts a real server via `go run . serve` against a temp directory,
then exercises admin CLI commands and HTTP API endpoints.

Usage:
    python tests/integration_test.py
"""

import http.client
import json
import os
import re
import shutil
import signal
import socket
import subprocess
import sys
import tempfile
import time

PROJECT_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# Shared state across tests (populated during test run)
PORT = None
DATA_DIR = None
RW_TOKEN = None
RO_TOKEN = None


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------


def find_free_port():
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as s:
        s.bind(("127.0.0.1", 0))
        return s.getsockname()[1]


def artf(*args):
    """Run an artf CLI command. Returns (exit_code, stdout, stderr)."""
    cmd = ["go", "run", ".", "--data", DATA_DIR] + list(args)
    result = subprocess.run(
        cmd,
        capture_output=True,
        text=True,
        timeout=30,
        cwd=PROJECT_ROOT,
    )
    return result.returncode, result.stdout, result.stderr


def api_request(method, path, token=None, body=None):
    """Send an HTTP request to the main API server.

    Returns (status_code, response_body_bytes).
    """
    conn = http.client.HTTPConnection("127.0.0.1", PORT, timeout=10)
    headers = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    # Use putrequest to avoid automatic URL encoding/normalization
    conn.putrequest(method, path, skip_host=False, skip_accept_encoding=False)
    for k, v in headers.items():
        conn.putheader(k, v)
    if body is not None:
        conn.putheader("Content-Length", str(len(body)))
    conn.endheaders(body)
    resp = conn.getresponse()
    data = resp.read()
    conn.close()
    return resp.status, data


def api_json(method, path, token=None, body=None):
    """Like api_request but parses JSON response body."""
    status, data = api_request(method, path, token, body)
    try:
        parsed = json.loads(data)
    except (json.JSONDecodeError, ValueError):
        parsed = None
    return status, parsed, data


def wait_for_server(port, timeout=30):
    """Poll GET /ping until the server is ready."""
    deadline = time.time() + timeout
    while time.time() < deadline:
        try:
            conn = http.client.HTTPConnection("127.0.0.1", port, timeout=2)
            conn.request("GET", "/ping")
            resp = conn.getresponse()
            resp.read()
            conn.close()
            if resp.status == 200:
                return
        except (ConnectionRefusedError, OSError, http.client.HTTPException):
            pass
        time.sleep(0.3)
    raise TimeoutError(f"Server did not start within {timeout}s")


def start_server(data_dir, port):
    proc = subprocess.Popen(
        ["go", "run", ".", "serve", "--data", data_dir, "--port", str(port)],
        cwd=PROJECT_ROOT,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
    )
    try:
        wait_for_server(port)
    except TimeoutError:
        rc = proc.poll()
        if rc is not None:
            _, stderr = proc.communicate(timeout=5)
            raise RuntimeError(
                f"Server exited with code {rc} before becoming ready.\n"
                f"stderr: {stderr.decode(errors='replace')}"
            )
        raise
    return proc


def stop_server(proc):
    if proc.poll() is not None:
        return
    proc.send_signal(signal.SIGTERM)
    try:
        proc.wait(timeout=5)
    except subprocess.TimeoutExpired:
        proc.kill()
        proc.wait(timeout=5)


def assert_eq(actual, expected, msg=""):
    if actual != expected:
        raise AssertionError(f"{msg}: expected {expected!r}, got {actual!r}")


def assert_in(needle, haystack, msg=""):
    if needle not in haystack:
        raise AssertionError(f"{msg}: {needle!r} not found in {haystack!r}")


def assert_not_in(needle, haystack, msg=""):
    if needle in haystack:
        raise AssertionError(f"{msg}: {needle!r} unexpectedly found in {haystack!r}")


def parse_repo_id(stdout):
    """Extract repo ID from 'Repo 'name' created (id: N)' output."""
    m = re.search(r"\(id:\s*(\d+)\)", stdout)
    if not m:
        raise AssertionError(f"Could not parse repo ID from: {stdout!r}")
    return m.group(1)


def parse_key_id(name, ls_stdout):
    """Extract key ID from key ls table output by matching the key name."""
    for line in ls_stdout.splitlines():
        if name in line:
            m = re.match(r"\s*(\d+)", line)
            if m:
                return m.group(1)
    raise AssertionError(f"Could not find key {name!r} in key ls output:\n{ls_stdout}")


# ---------------------------------------------------------------------------
# Tests
# ---------------------------------------------------------------------------


def test_ping_and_version():
    # HTTP ping
    status, body = api_request("GET", "/ping")
    assert_eq(status, 200, "ping status")
    assert_in(b"pong", body, "ping body")

    # HTTP version
    status, parsed, _ = api_json("GET", "/version")
    assert_eq(status, 200, "version status")
    assert isinstance(parsed, dict), f"version response is not a dict: {parsed}"

    # CLI status
    rc, stdout, stderr = artf("status")
    assert_eq(rc, 0, f"artf status exit code (stderr: {stderr})")
    assert_in("Running version", stdout, "artf status output")


def test_admin_key_crud():
    global RW_TOKEN, RO_TOKEN

    # Create read-write key
    rc, stdout, stderr = artf(
        "key", "new", "--name", "rw-key", "--readonly=false", "-q"
    )
    assert_eq(rc, 0, f"create rw key (stderr: {stderr})")
    RW_TOKEN = stdout.strip()
    assert_in("artf_", RW_TOKEN, "rw key prefix")

    # Create read-only key
    rc, stdout, stderr = artf("key", "new", "--name", "ro-key", "--readonly", "-q")
    assert_eq(rc, 0, f"create ro key (stderr: {stderr})")
    RO_TOKEN = stdout.strip()
    assert_in("artf_", RO_TOKEN, "ro key prefix")

    # List keys — should have 2
    rc, stdout, _ = artf("key", "ls")
    assert_eq(rc, 0, "key ls exit code")
    assert_in("rw-key", stdout, "key ls contains rw-key")
    assert_in("ro-key", stdout, "key ls contains ro-key")

    # Create a temp key, then delete it
    rc, stdout, _ = artf("key", "new", "--name", "temp-key", "--readonly=false", "-q")
    assert_eq(rc, 0, "create temp key")

    rc, ls_out, _ = artf("key", "ls")
    assert_eq(rc, 0, "key ls for temp key")
    assert_in("temp-key", ls_out, "temp key in list")
    temp_id = parse_key_id("temp-key", ls_out)

    rc, stdout, _ = artf("key", "rm", temp_id, "--yes")
    assert_eq(rc, 0, "delete temp key")
    assert_in("Key deleted", stdout, "key rm output")

    # Verify temp key is gone
    rc, stdout, _ = artf("key", "ls")
    assert_eq(rc, 0, "key ls after delete")
    assert_not_in("temp-key", stdout, "temp key should be gone")


def test_admin_repo_crud():
    # Create repo
    rc, stdout, stderr = artf(
        "repo", "new", "--name", "test-repo", "--keep-count", "5", "--keep-days", "30"
    )
    assert_eq(rc, 0, f"create test-repo (stderr: {stderr})")
    assert_in("created", stdout, "repo new output")
    repo_id = parse_repo_id(stdout)

    # List repos
    rc, stdout, _ = artf("repo", "ls")
    assert_eq(rc, 0, "repo ls")
    assert_in("test-repo", stdout, "repo ls contains test-repo")

    # Repo info
    rc, stdout, _ = artf("repo", "info", repo_id)
    assert_eq(rc, 0, "repo info")
    assert_in("test-repo", stdout, "repo info name")
    assert_in("KeepCount:     5", stdout, "repo info keep count")
    assert_in("KeepDays:      30", stdout, "repo info keep days")

    # Update repo (must pass both flags to avoid interactive prompts)
    rc, stdout, stderr = artf(
        "repo", "edit", repo_id, "--keep-count", "10", "--keep-days", "30"
    )
    assert_eq(rc, 0, f"repo edit (stderr: {stderr})")
    assert_in("updated", stdout, "repo edit output")

    # Verify update
    rc, stdout, _ = artf("repo", "info", repo_id)
    assert_eq(rc, 0, "repo info after update")
    assert_in("KeepCount:     10", stdout, "repo info updated keep count")
    assert_in("KeepDays:      30", stdout, "repo info keep days unchanged")

    # Create and delete a temp repo
    rc, stdout, _ = artf(
        "repo", "new", "--name", "temp-repo", "--keep-count", "0", "--keep-days", "0"
    )
    assert_eq(rc, 0, "create temp repo")
    temp_id = parse_repo_id(stdout)

    rc, stdout, _ = artf("repo", "rm", temp_id, "--yes")
    assert_eq(rc, 0, "delete temp repo")
    assert_in("Repo deleted", stdout, "repo rm output")

    rc, stdout, _ = artf("repo", "ls")
    assert_not_in("temp-repo", stdout, "temp repo should be gone")


def test_auth_rejection():
    # No auth
    status, body = api_request("GET", "/test-repo")
    assert_eq(status, 401, "no auth status")
    assert_in(b"unauthorized", body, "no auth body")

    # Bad token
    status, body = api_request("GET", "/test-repo", token="invalid_token_here")
    assert_eq(status, 401, "bad token status")
    assert_in(b"unauthorized", body, "bad token body")

    # No auth on upload
    status, body = api_request("PUT", "/test-repo/foo.tar.gz", body=b"data")
    assert_eq(status, 401, "no auth upload status")

    # Bad token on upload
    status, body = api_request(
        "PUT", "/test-repo/foo.tar.gz", token="invalid_token_here", body=b"data"
    )
    assert_eq(status, 401, "bad token upload status")
    assert_in(b"unauthorized", body, "bad token upload body")

    # Upload with RO token
    status, body = api_request(
        "PUT", "/test-repo/foo.tar.gz", token=RO_TOKEN, body=b"data"
    )
    assert_eq(status, 403, "ro token upload status")
    assert_in(b"forbidden", body, "ro token upload body")

    # Valid token should work
    status, body = api_request("GET", "/test-repo", token=RW_TOKEN)
    assert_eq(status, 200, "valid token status")


def test_upload_download_list():
    # Upload two artifacts
    status, parsed, _ = api_json(
        "PUT", "/test-repo/artifact-v1.tar.gz", token=RW_TOKEN, body=b"content-v1"
    )
    assert_eq(status, 201, "upload v1 status")
    assert_eq(parsed["name"], "artifact-v1.tar.gz", "upload v1 name")

    status, parsed, _ = api_json(
        "PUT", "/test-repo/artifact-v2.tar.gz", token=RW_TOKEN, body=b"content-v2"
    )
    assert_eq(status, 201, "upload v2 status")
    assert_eq(parsed["name"], "artifact-v2.tar.gz", "upload v2 name")

    # List artifacts
    status, parsed, _ = api_json("GET", "/test-repo", token=RW_TOKEN)
    assert_eq(status, 200, "list artifacts status")
    assert_eq(len(parsed), 2, "artifact count")
    names = {a["name"] for a in parsed}
    assert_in("artifact-v1.tar.gz", names, "v1 in list")
    assert_in("artifact-v2.tar.gz", names, "v2 in list")

    # Download and verify content
    status, body = api_request("GET", "/test-repo/artifact-v1.tar.gz", token=RW_TOKEN)
    assert_eq(status, 200, "download v1 status")
    assert_eq(body, b"content-v1", "download v1 content")

    status, body = api_request("GET", "/test-repo/artifact-v2.tar.gz", token=RW_TOKEN)
    assert_eq(status, 200, "download v2 status")
    assert_eq(body, b"content-v2", "download v2 content")


def test_bad_auth_download():
    # Attempt to download with no auth
    status, body = api_request("GET", "/test-repo/artifact-v1.tar.gz")
    assert_eq(status, 401, "no auth download status")
    assert_in(b"unauthorized", body, "no auth download body")

    # Attempt to download with bad token
    status, body = api_request(
        "GET", "/test-repo/artifact-v1.tar.gz", token="invalid_token_here"
    )
    assert_eq(status, 401, "bad token download status")
    assert_in(b"unauthorized", body, "bad token download body")


def test_latest_download():
    status, body = api_request("GET", "/test-repo/latest", token=RW_TOKEN)
    assert_eq(status, 200, "latest download status")
    assert_eq(body, b"content-v2", "latest should be v2")


def test_duplicate_upload():
    status, parsed, _ = api_json(
        "PUT", "/test-repo/artifact-v1.tar.gz", token=RW_TOKEN, body=b"duplicate"
    )
    assert_eq(status, 409, "duplicate upload status")
    assert_in("already exists", parsed["error"], "duplicate error message")


def test_readonly_key_upload():
    # Read operations with ro key should work
    status, _ = api_request("GET", "/test-repo", token=RO_TOKEN)
    assert_eq(status, 200, "ro key list status")

    status, _ = api_request("GET", "/test-repo/artifact-v1.tar.gz", token=RO_TOKEN)
    assert_eq(status, 200, "ro key download status")

    # Write with ro key should be forbidden
    status, body = api_request(
        "PUT", "/test-repo/new-artifact.tar.gz", token=RO_TOKEN, body=b"data"
    )
    assert_eq(status, 403, "ro key upload status")
    assert_in(b"forbidden", body, "ro key upload body")


def test_validation_errors():
    # Reserved name "latest"
    status, parsed, _ = api_json("PUT", "/test-repo/latest", token=RW_TOKEN, body=b"x")
    assert_eq(status, 400, "latest reserved status")
    assert_in("latest", parsed["error"], "latest error message")

    # Note: /test-repo/.. and /test-repo/. get cleaned by Go's HTTP mux
    # before reaching the handler, so they can't be tested via HTTP.
    # The validate package itself is tested in unit tests.

    # Invalid characters (use chars that are valid in URLs but fail artf validation)
    status, _, _ = api_json("PUT", "/test-repo/bad@name", token=RW_TOKEN, body=b"x")
    assert_eq(status, 400, "invalid chars artifact status")

    # Too long name (65 chars, max is 64)
    long_name = "a" * 65
    status, _, _ = api_json("PUT", f"/test-repo/{long_name}", token=RW_TOKEN, body=b"x")
    assert_eq(status, 400, "too long artifact name status")

    # Invalid repo names via CLI
    rc, _, stderr = artf("repo", "new", "--name", "..")
    assert rc != 0, f"dotdot repo name should fail (stderr: {stderr})"

    rc, _, stderr = artf("repo", "new", "--name", "bad name!")
    assert rc != 0, f"invalid repo name should fail (stderr: {stderr})"

    # Nonexistent repo via API
    status, parsed, _ = api_json("GET", "/nonexistent-repo", token=RW_TOKEN)
    assert_eq(status, 404, "nonexistent repo status")
    assert_eq(parsed["error"], "repo not found", "nonexistent repo error")

    # Nonexistent artifact via API
    status, parsed, _ = api_json(
        "GET", "/test-repo/nonexistent-artifact", token=RW_TOKEN
    )
    assert_eq(status, 404, "nonexistent artifact status")
    assert_eq(parsed["error"], "artifact not found", "nonexistent artifact error")


def test_repo_delete_cascades():
    # Create a repo and upload an artifact
    rc, stdout, stderr = artf(
        "repo", "new", "--name", "delete-me", "--keep-count", "0", "--keep-days", "0"
    )
    assert_eq(rc, 0, f"create delete-me repo (stderr: {stderr})")
    repo_id = parse_repo_id(stdout)

    status, _, _ = api_json(
        "PUT", "/delete-me/file.bin", token=RW_TOKEN, body=b"deletable"
    )
    assert_eq(status, 201, "upload to delete-me")

    # Verify artifact shows in repo info
    rc, stdout, _ = artf("repo", "info", repo_id)
    assert_eq(rc, 0, "repo info delete-me")
    assert_in("file.bin", stdout, "artifact in repo info")

    # Delete the repo
    rc, stdout, _ = artf("repo", "rm", repo_id, "--yes")
    assert_eq(rc, 0, "delete repo")
    assert_in("Repo deleted", stdout, "repo rm output")

    # API should return 404 for the deleted repo
    status, _, _ = api_json("GET", "/delete-me", token=RW_TOKEN)
    assert_eq(status, 404, "deleted repo API status")

    # CLI should fail for the deleted repo
    rc, _, _ = artf("repo", "info", repo_id)
    assert rc != 0, "repo info should fail after delete"

    # Directory should be gone
    repo_dir = os.path.join(DATA_DIR, "repos", "delete-me")
    assert not os.path.exists(repo_dir), f"repo directory should be removed: {repo_dir}"


# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------


def main():
    global PORT, DATA_DIR

    DATA_DIR = tempfile.mkdtemp(prefix="artf-integration-")
    PORT = find_free_port()
    proc = None
    passed = 0
    failed = 0
    errors = []

    tests = [
        test_ping_and_version,
        test_admin_key_crud,
        test_admin_repo_crud,
        test_auth_rejection,
        test_upload_download_list,
        test_bad_auth_download,
        test_latest_download,
        test_duplicate_upload,
        test_readonly_key_upload,
        test_validation_errors,
        test_repo_delete_cascades,
    ]

    try:
        print(f"Starting server on port {PORT} with data dir {DATA_DIR}")
        proc = start_server(DATA_DIR, PORT)
        print(f"Server ready (pid {proc.pid})\n")

        for test_fn in tests:
            name = test_fn.__name__
            try:
                test_fn()
                print(f"  PASS  {name}")
                passed += 1
            except Exception as e:
                print(f"  FAIL  {name}: {e}")
                failed += 1
                errors.append((name, e))

    except Exception as e:
        print(f"Setup error: {e}")
        failed += 1
        errors.append(("setup", e))

    finally:
        if proc:
            stop_server(proc)
        shutil.rmtree(DATA_DIR, ignore_errors=True)

    print(f"\n{passed} passed, {failed} failed")
    if errors:
        print("\nFailures:")
        for name, err in errors:
            print(f"  {name}: {err}")
    sys.exit(1 if failed else 0)


if __name__ == "__main__":
    main()
