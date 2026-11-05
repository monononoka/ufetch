# ufetch

A lightweight Go command-line downloader with a protocol-resolver architecture. The transfer engine and API integrations use only the Go standard library.

## Build and use

```powershell
go build -o ufetch.exe .
.\ufetch.exe -h
.\ufetch.exe https://example.com/file.zip
.\ufetch.exe -dir .\downloads https://example.com/file.zip
.\ufetch.exe -o .\downloads\file.zip https://example.com/file.zip
.\ufetch.exe app:cli/cli
.\ufetch.exe hf:openai-community/gpt2
.\ufetch.exe npm:lodash@4.17.21
.\ufetch.exe pypi:requests
```

- `app:<owner/repository>` selects a compatible asset from the repository's latest stable GitHub Release.
- `app:<search terms>` searches GitHub repositories and uses the top result. Use `owner/repository` to specify an exact project.
- `hf:<search terms>` searches Hugging Face models and uses the most-downloaded matching model.
- `hf:<owner/model>` resolves an exact model repository. If it contains multiple supported weight files, ufetch displays a numbered selection menu.
- `hf:<owner/model/path/to/file>` selects a specific repository file.
- `npm:<package>` or `npm:<package>@<version>` downloads a public npm package archive.
- `pypi:<project>` or `pypi:<project>@<version>` lists public Python package distributions for selection.

The Hugging Face integration downloads one file per invocation. If a model uses multiple weight shards, download each required shard separately. Private or gated repositories are not supported yet. GitHub API requests are unauthenticated and subject to GitHub's rate limits.

The npm and PyPI protocols use their public package registries and download archives only; ufetch does not install or execute packages. Registry availability, package licenses, and local laws can vary. Check that each package is lawful and appropriate for your use; ufetch does not verify ownership or licensing.

Downloads are written to `<destination>.part` until complete, then renamed to the destination. If interrupted, partial data is retained. On the next run, ufetch asks the server to continue with an HTTP `Range` request. If the server ignores that request and returns the whole file, ufetch safely starts that transfer from the beginning. Resuming also checks a small `<destination>.part.url` sidecar so data from a different URL is never appended accidentally.

The server must support HTTP byte ranges to resume a single file from its previous offset. Without that support, ufetch can retain partial bytes, but a retry has to download that resource from the beginning. The current implementation uses ranges for resuming an interrupted transfer; parallel multi-range chunk downloads are not implemented yet. ufetch does not host files or require its own server.

Use `-dir DIR` to save the selected file in a directory while keeping its source filename. Use `-o FILE` for a specific path; the two options cannot be combined. The terminal shows an ASCII progress bar when the server provides the total file size.

## Development

```sh
go test ./...
go vet ./...
go build -o ufetch.exe .
```

GitHub Actions runs tests, vet, and cross-build checks on pushes and pull requests. Pushing a version tag such as `v0.1.0` triggers a release workflow that builds Windows, Linux, and macOS binaries and attaches them to a GitHub Release.

## License

Licensed under the GNU General Public License, version 3. See [LICENSE](LICENSE).
