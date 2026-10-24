# Changelog

All notable changes to this project are documented here. Versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased] - 1.1.0

### Upgrading

These changes can break existing setups:

- **`zipreport debug` binds to `127.0.0.1` by default.** In Docker or for remote access, pass `--host 0.0.0.0`; that also turns off the Host-header check.
- **`WithSecureSSL(true)` now verifies certificates.** Self-signed or private-CA servers fail with x509 errors; pass a client with `RootCAs` through `WithHTTPClient`. Verification is still skipped by default, and `WithSecureSSL` has no effect when you supply your own client.
- **Builds fail on any per-file error** and return a nil ZipFs (`BuildZipFs`/`Build`, `zipreport build`). These are now fatal:
  - unreadable files;
  - a symlinked `index.html`/`manifest.json`.
  - With `-s`: broken or looping symlinks, symlinked folders, and symlinks that point outside the template folder.
- **A malformed or non-object `data.json` fails every render** (`invalid data.json`). An empty file, `null`, or a file with a UTF-8 byte-order mark is accepted.
- **Folder templates are contained.** No read or write may leave the folder, whether through `..` or a symlink that resolves outside it. `LoadDir` names a required file that is such a symlink (`fileutils.ErrOutsideFolder`). Includes and assets that resolve outside are not found, and `Save()` skips them.
- **`manifest.json` `report` must be a relative path inside the report.** A leading `/` is ignored.
- **Unportable file names fail.** In folders, `Save()`, renders and builds fail and name the file; in `.zpt` files, loading fails (`fileutils.ErrUnportableName`). An unportable name has a `..` segment (between `/` or `\`), a drive-letter prefix such as `a:`, or a leading `\`.
- **Size and count limits:**
  - **`.zpt` loads** fail with `fileutils.ErrZipTooLarge`:
    - over 512 MiB uncompressed (raise it with `fileutils.MaxUncompressedSize`);
    - more than 10,000 entries;
    - an entry name longer than 1,024 bytes;
    - nesting deeper than 32 folders.
  - **Folder renders and `Save()`** fail with `ErrZipTooLarge`:
    - past 10,000 files and folders (a folder reached through several symlinks counts once per path);
    - past 100,000 listed directory entries;
    - past `MaxUncompressedSize` bytes;
    - nesting deeper than 32 levels.
  - **PDF responses** over 512 MiB fail with `processor.ErrResponseTooLarge` (raise it with `WithMaxResponseSize`; `api.RenderPDF` cannot change either limit).
- **MIME headers:** CR, LF, NUL or other control characters in Subject/From/To are an error, as is an ASCII subject over 989 characters.

### Changed

- **API renders never modify the loaded report.**
  - **How:** each render writes into a private copy (`ReportFile.Clone()`, an in-memory overlay), so a loaded report can be shared across goroutines.
  - **Data:** the caller's data map is not modified (top level only; templates can still assign into nested values).
  - **Filters:** a wrapper filter that writes into a captured `ReportFile` writes into the original.
- **Folder `Save()` and `BuildFromReportFile`:**
  - ship every non-hidden file in the folder on every render;
  - skip root-level `_gen_*` files;
  - ship the on-disk `manifest.json` (edits made through `GetManifest()` are not sent);
  - fail without `index.html`.
- **`.zpt` output** is deterministic within one Go toolchain.
- **ServerBackend:**
  - option order no longer matters;
  - `WithHTTPClient(c)` combined with `WithTimeout` uses a copy of `c`;
  - `WithHTTPClient(nil)` falls back to the default client;
  - the content type is checked before the body is read;
  - error bodies are bounded and truncated to 1 KiB.
- **MIME:**
  - base64 lines are wrapped at 76 characters;
  - non-ASCII subjects are RFC 2047-encoded, and `To` is folded;
  - only `src` and `<link href>` files that exist in the report are embedded, and never `.html` files or hidden files;
  - `<a href>` is left untouched;
  - line breaks in the title are collapsed;
  - the report file named in the manifest is used.
- **Debug server:**
  - when bound to loopback, it answers 403 to requests whose Host is not localhost;
  - `ReadHeaderTimeout` is 10s;
  - dotfiles are never served.
- **Filters and errors:** errors from filter registration are returned.
- **Tooling:**
  - CI tests Go 1.24.x, oldstable and stable;
  - lint is pinned to golangci-lint v2.12.2 (a v2 config) on Go 1.26.x;
  - release builds use `stable`;
  - `make test` and CI run `./...`, including CLI end-to-end tests (`-short` skips them).

### Added

- `fileutils`:
  - `OverlayFs`, `NewOverlayFs`, `ErrReadOnly`;
  - `(*DiskFs).Snapshot`, `(*ZipFs).Clone`;
  - `MaxUncompressedSize`, `ErrZipTooLarge`, `ErrOutsideFolder`, `ErrUnportableName`;
  - `IsHiddenPath`, `IsUnportableName`.
- `report`: `(*ReportFile).Clone`, `(*ReportJob).WithReport`.
- `processor`: `WithMaxResponseSize`, `ErrResponseTooLarge`.
- CLI: `zipreport debug --host`.

## [1.0.1] and earlier

See the [GitHub releases](https://github.com/zipreport/zipreport-go/releases).
