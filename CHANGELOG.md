# Changelog

All notable changes to this project are documented here. Versions follow [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Upgrading

- **The default client refuses redirects to another host.** A deployment where zipreport-server answers with a redirect to another host, another port, or another spelling of the hostname (including an explicit default port such as `:80`) now fails with `processor.ErrRedirectRefused`. Point the URL at the final host.
- **`api.RenderPDF` has a new signature** (`opts ...processor.ServerOption`). Calls still compile; code that stores `api.RenderPDF` in a variable whose type spells out the old func type does not.

### Security

- The default client refuses redirects to another host or from https to http (`processor.ErrRedirectRefused`). Before, it sent `X-Auth-Key`, and on 307/308 the report, to the redirect target. Clients passed with `WithHTTPClient` are unchanged; set `CheckRedirect: processor.CheckRedirect` on them.
- **The debug server and MIME embedding check the file actually read, for folder templates.** A symlink to a dotfile (config.txt -> .env, assets -> .git), a request for //index.html, and on case-insensitive or Windows disks another spelling of a file name (INDEX.HTML, ENV~1) no longer serve or embed hidden files or the raw template. .zpt files are still judged by name.

### Changed

- For folder templates, a MIME reference whose case differs from the file name (macOS, Windows) or that uses backslashes (Windows) is no longer embedded, and a symlink into a hidden folder is no longer served or embedded.

### Added

- `RenderContext` / `RenderDefaultsContext` / `ZipReportProcessor.ProcessContext` / `ServerBackend.RenderContext` / `processor.ContextRenderBackend` / `processor.CheckRedirect`.
- `api.RenderPDF` accepts `...processor.ServerOption`.
- `fileutils.IsServable`.

### Fixed

- A trailing `/` in the server URL no longer produces `//v2/render`.

## [1.1.1] - 2026-10-25

### Fixed

- **`zipreport version` reports the real version.** Builds without `-ldflags`, such as `go install`, used to print a hard-coded `1.0.0`; they now print the module version Go recorded (without the leading `v`), or `dev` when there is none. Builds that set `-X main.version=...` are unchanged.
- **Docs:**
  - image filters take positional arguments; keyword arguments were documented but are ignored;
  - use `| json | safe` inside `<script>`;
  - corrected the `version` command, the `ZIPREPORT_KEY` variable, and the Go 1.24 minimum.

## [1.1.0] - 2026-10-25

### Upgrading

These changes can break existing setups. [MIGRATION.md](MIGRATION.md) explains what to do about each.

- **`zipreport debug` binds to `127.0.0.1` by default.** In Docker or for remote access, pass `--host 0.0.0.0`; that also turns off the Host-header check.
- **`WithSecureSSL(true)` now verifies certificates.** Self-signed or private-CA servers fail with x509 errors; pass a client with `RootCAs` through `WithHTTPClient`. Verification is still skipped by default, and `WithSecureSSL` has no effect when you supply your own client.
- **Builds fail on any per-file error** and return a nil ZipFs (`BuildZipFs`/`Build`, `zipreport build`). These are now fatal:
  - unreadable files;
  - a symlinked `index.html`/`manifest.json`.
  - With `-s`: broken or looping symlinks, symlinked folders, and symlinks that point outside the template folder.
- **A `MiyaRender` renders once.** A second `Render`, `RenderTemplate` or `RenderString` on the same renderer returns `template.ErrRendererUsed`. Before, rendering the same template twice returned its raw source; other repeat calls (several `RenderString`s, `RenderString` then `Render`, or a retry after a missing-parameter error) worked and now fail too. Create a new renderer per render, as `api.ZipReport` already does.
- **API renders no longer write into your `ReportFile`.** After `api.ZipReport` or `api.MIMEReport` renders, `report.html` and generated images are not in the `ReportFile` you passed in; use the `JobResult`.
- **A malformed or non-object `data.json` fails every render** (`invalid data.json`). An empty file, `null`, or a file with a UTF-8 byte-order mark is accepted.
- **Folder templates are contained.** No read or write may leave the folder, whether through `..` or a symlink that resolves outside it. `LoadDir` names a required file that is such a symlink (`fileutils.ErrOutsideFolder`). Includes and assets that resolve outside are not found, and `Save()` skips them.
- **`manifest.json` `report` must be a relative path inside the report.** A leading `/` is ignored.
- **Unportable file names fail.** In folders, `Save()`, renders and builds fail and name the file; in `.zpt` files, loading fails (`fileutils.ErrUnportableName`). `ZipFs.Add`/`Mkdir`, `OverlayFs.Add` and `ReportFile.Add` return `ErrUnportableName` immediately, and `ErrInvalidPath` for an empty name or `.`. An unportable name has a `..` segment (between `/` or `\`), a drive-letter prefix such as `a:`, or a leading `\`.
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
  - **PDF responses** over 512 MiB fail with `processor.ErrResponseTooLarge` (raise it with `WithMaxResponseSize`; `api.RenderPDF` takes no options, so its limit is fixed).
- **MIME resource paths** are resolved relative to the folder of the manifest's report file; a leading `/` means the report root. Before, they were looked up from the root as written.
- **MIME headers:** CR, LF, NUL or other control characters in Subject/From/To are an error, as is an ASCII subject over 989 characters.
- **Templates render with miya v0.2.0, which changes number handling.**
  - Numbers from JSON data now compare correctly with template literals; `{% if total > 0 %}` with `total` = 0 used to take the true branch.
  - Comparing a string with a number using `<`, `>`, `<=` or `>=` now fails the render, so numbers must arrive as numbers.
  - Numbers are no longer printed with thousands separators or rounded to two decimals (`10023`, not `10,023`; `3.14159`, not `3.14`), and arithmetic results print in full (`0.1 + 0.2` prints `0.30000000000000004`). Use `|format_number` or `|currency` where you want formatting.
  - A template whose includes and macro calls nest more than 200 levels deep (combined), such as one that includes itself, now fails with an error instead of crashing the process.

### Changed

- **API renders never modify the loaded report.**
  - **How:** each render writes into a private copy (`ReportFile.Clone()`, an in-memory overlay), so a loaded report can be shared across goroutines.
  - **Data:** the caller's data map is not modified (top level only; templates can still assign into nested values).
  - **Filters:** a wrapper filter that writes into a captured `ReportFile` writes into the original.
- **Folder `Save()` and `BuildFromReportFile`:**
  - ship every non-hidden file in the folder on every render;
  - skip root-level `_gen_*` files;
  - ship the on-disk `manifest.json` (edits made through `GetManifest()` are not sent);
  - fail without `index.html`;
  - skip unreadable files and folders;
  - work with custom `FsInterface` implementations, whose `List` methods must return base names.
- **Builds** follow a symlinked source folder, accept a source folder whose name starts with `.`, and skip sockets, FIFOs and devices.
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
  - the report file named in the manifest is used;
  - Subject, From and To are trimmed, and a blank subject falls back to the title;
  - a file referenced several times is embedded once;
  - non-ASCII attachment names are RFC 2231-encoded;
  - references with any URL scheme are skipped.
- **Debug server:**
  - when bound to loopback, it answers 403 to requests whose Host is not localhost;
  - `ReadHeaderTimeout` is 10s;
  - dotfiles are never served;
  - renders use an in-memory copy, so nothing is written into the template folder;
  - the startup message shows the bind address.
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
