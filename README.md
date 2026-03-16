# artf

A minimal artifact management system for home use. Store and retrieve build artifacts over HTTP with API key authentication.

## Quick start

```bash
make build
```

### 1. Start the server

```bash
artf serve
```

This starts the API server on `127.0.0.1:8365` and an admin server on a Unix socket at `~/.artf/artf.sock`.

Data is stored in `~/.artf/` by default, can be changed with ARTF_DATA env variable or `--data` when starting the server.

Uploads are limited to `100m` by default. Override that with `--max-upload-size` or `ARTF_MAX_UPLOAD_SIZE`. Accepted suffixes are `k`, `m`, and `g`.

### 2. Create a repo

```bash
artf repo new --name my-builds --keep-count 10 --keep-days 30
```

`--keep-count` and `--keep-days` control artifact retention (0 = keep all).

### 3. Edit repo settings

```bash
artf repo edit my-builds --keep-count 20 --keep-days 60
```

Both flags are optional — omitted flags will prompt interactively with current values pre-filled.

### 4. Create an API key

```bash
artf key new --name ci --readonly=false
```

Save the key — it is only shown once.

### 5. Upload and download artifacts

```bash
# Upload
curl -X PUT -H "Authorization: Bearer artf_..." \
  --data-binary @build.tar.gz \
  http://127.0.0.1:8365/my-builds/build.tar.gz

# Download
curl -H "Authorization: Bearer artf_..." \
  http://127.0.0.1:8365/my-builds/build.tar.gz -o build.tar.gz

# List artifacts in a repo
curl -H "Authorization: Bearer artf_..." \
  http://127.0.0.1:8365/my-builds
```

Uploading the same artifact name twice returns `409 Conflict`. Uploads larger than the configured limit return `413 Payload Too Large`.

## CLI reference

| Command | Description |
|---|---|
| `artf serve` | Start the server |
| `artf repo new` | Create a repository |
| `artf repo ls` | List repositories |
| `artf repo info <name>` | Show repository details |
| `artf repo edit [id or name]` | Edit repo retention settings |
| `artf repo rm <name>` | Delete a repository |
| `artf key new` | Create an API key |
| `artf key ls` | List API keys |
| `artf key rm <name>` | Delete an API key |
| `artf status` | Show server status |
| `artf version` | Show version info |

## Configuration

| Flag / Env var | Default | Description |
|---|---|---|
| `-d` / `ARTF_DATA` | `~/.artf` | Data directory |
| `-v` | off | Verbose logging |

The serve command also accepts:

| Flag / Env var | Default | Description |
|---|---|---|
| `-H` / `--host` | `127.0.0.1` | Listen address |
| `-P` / `--port` | `8365` | Listen port |
| `--systemd` | off | Use systemd socket activation |
| `--max-upload-size` / `ARTF_MAX_UPLOAD_SIZE` | `100m` | Maximum upload body size; accepts bytes or `k`, `m`, `g` suffixes |

## License

See [LICENSE](LICENSE).
