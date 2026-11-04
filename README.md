# ufetch

A lightweight Go command-line downloader with a protocol-resolver architecture. The transfer engine and API integrations use only the Go standard library.

## Build and use

```powershell
go build -o ufetch.exe .
.\ufetch.exe https://example.com/file.zip
.\ufetch.exe -o .\downloads\file.zip https://example.com/file.zip
.\ufetch.exe app:cli/cli
.\ufetch.exe hf:openai-community/gpt2
```

- `app:<owner/repository>` selects a compatible asset from the repository's latest stable GitHub Release.
- `app:<search terms>` searches GitHub repositories and uses the top result. Use `owner/repository` to specify an exact project.
- `hf:<search terms>` searches Hugging Face models and uses the most-downloaded matching model.
- `hf:<owner/model>` resolves an exact model repository. If it contains multiple supported weight files, ufetch displays a numbered selection menu.
- `hf:<owner/model/path/to/file>` selects a specific repository file.

The Hugging Face integration downloads one file per invocation. If a model uses multiple weight shards, download each required shard separately. Private or gated repositories are not supported yet. GitHub API requests are unauthenticated and subject to GitHub's rate limits.

Downloads are written to `<destination>.part` until complete, then renamed to the destination. If interrupted, partial data is retained. On the next run, ufetch asks the server to continue with an HTTP `Range` request. If the server ignores that request and returns the whole file, ufetch safely starts that transfer from the beginning. Resuming also checks a small `<destination>.part.url` sidecar so data from a different URL is never appended accidentally.

The server must support HTTP byte ranges to resume a single file from its previous offset. Without that support, ufetch can retain partial bytes, but a retry has to download that resource from the beginning. ufetch does not host files or require its own server.

## Development

```sh
go test ./...
go vet ./...
go build -o ufetch.exe .
```

GitHub Actions runs tests, vet, and cross-build checks on pushes and pull requests. Pushing a version tag such as `v0.1.0` triggers a release workflow that builds Windows, Linux, and macOS binaries and attaches them to a GitHub Release.

## License

Licensed under the GNU General Public License, version 3. See [LICENSE](LICENSE).
