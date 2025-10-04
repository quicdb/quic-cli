# QuicDB CLI

Official CLI for [QuicDB](https://quicdb.com) to manage your database branches.

## Installation

### macOS and Linux (Homebrew)

```bash
brew tap quicdb/tap
brew install quic
```

### Manual Installation

Download the latest binary for your platform from [GitHub Releases](https://github.com/quicdb/quic-cli/releases):

```bash
# macOS (Apple Silicon)
curl -L https://github.com/quicdb/quic-cli/releases/latest/download/quic-darwin-arm64 -o quic
chmod +x quic
sudo mv quic /usr/local/bin/

# macOS (Intel)
curl -L https://github.com/quicdb/quic-cli/releases/latest/download/quic-darwin-amd64 -o quic
chmod +x quic
sudo mv quic /usr/local/bin/

# Linux (x86_64)
curl -L https://github.com/quicdb/quic-cli/releases/latest/download/quic-linux-amd64 -o quic
chmod +x quic
sudo mv quic /usr/local/bin/
```

Verify the installation:

```bash
quic version
```

## Getting Started

### Authentication

Login to your QuicDB account:

```bash
quic login
```

This will open your browser to authenticate.

#### For machine-to-machine (M2M) authentication:

Create a service account in your QuicDB account and use the credentials:

```bash
quic login --client-id=<client_id> --client-secret=<client_secret>
```

### Managing Database Branches

**Create a branch:**

Outputs a connection string to your branch.

```bash
quic checkout my-feature
```

**List all branches:**

```bash
quic ls
```

**Delete a branch:**

```bash
quic delete my-feature
```

## Security

The QuicDB CLI stores authentication tokens securely using your operating system's credential manager:
- **macOS**: Keychain
- **Linux**: Secret Service (gnome-keyring, kwallet)

## Support

- **Issues**: https://github.com/quicdb/quic-cli/issues

## License

See [LICENSE](LICENSE) for details.
