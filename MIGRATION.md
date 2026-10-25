# Migration guide

This guide covers upgrading between ZipReport Go releases. It lists every behaviour change that can break an existing setup and says what to do about it. For the full list of changes, including additions, see [CHANGELOG.md](CHANGELOG.md).

## 1.0.x to 1.1.0

Most applications upgrade without code changes. Check the table below; each row links to the details.

| If you... | Then |
|---|---|
| run `zipreport debug` in Docker or open it from another machine | pass `--host 0.0.0.0` ([details](#debug-server-listens-on-localhost-only)) |
| use `WithSecureSSL(true)` with a self-signed or private-CA server | supply a client that trusts your CA ([details](#withsecuressltrue-verifies-certificates)) |
| build templates that contain unreadable files or symlinks | fix or remove them ([details](#builds-fail-on-any-file-error)) |
| ship a `data.json` that is not a JSON object | make it an object, or empty ([details](#datajson-must-be-a-json-object)) |
| use folder templates with symlinks or `..` paths that leave the folder | move the files into the folder ([details](#folder-templates-are-contained)) |
| set `report` in `manifest.json` | make it a relative path inside the report ([details](#manifestjson-report-must-be-a-relative-path)) |
| have file names with `..`, `a:` or a leading `\` | rename them ([details](#unportable-file-names-fail)) |
| load very large `.zpt` files or folders, or receive PDFs over 512 MiB | raise the limits ([details](#size-and-count-limits)) |
| put user input in MIME Subject, From or To | strip control characters first ([details](#mime-headers-reject-control-characters)) |
| read `report.html` or generated images from your `ReportFile` after an API render | use the `JobResult` instead ([details](#renders-no-longer-write-into-your-reportfile)) |
| call `ZipFs.Add`, `Mkdir` or `ReportFile.Add` with names you build yourself | handle the new error ([details](#unportable-file-names-fail)) |
| send MIME email from a report whose entry file is in a subfolder | check resource paths ([details](#mime-resource-paths-are-relative-to-the-report-file)) |
| write into a captured `ReportFile` from a custom filter, or edit the manifest in memory before rendering | read [Renders work on a copy](#renders-work-on-a-copy-of-the-report) |
| call `Render`, `RenderTemplate` or `RenderString` more than once on one `template.MiyaRender` | create a new renderer per render ([details](#a-miyarender-renders-once)) |

### Breaking changes

#### Debug server listens on localhost only

`zipreport debug` now binds to `127.0.0.1` by default. Before, it listened on all interfaces.

- **Docker, a VM, or access from another machine:** pass `--host 0.0.0.0`.
- **Loopback binds** (the default, `localhost`, `::1`) answer `403 Forbidden` to requests whose `Host` header is not a localhost name. This blocks DNS-rebinding attacks from web pages. Binding to any other address, such as `0.0.0.0`, turns the check off.
- Dotfiles are never served.

```sh
zipreport debug --host 0.0.0.0 ./my-template
```

#### `WithSecureSSL(true)` verifies certificates

In 1.0.x, `WithSecureSSL(true)` did not turn on certificate verification. It now does, so a server with a self-signed or private-CA certificate fails with an `x509` error.

The default is unchanged: without `WithSecureSSL(true)`, verification is skipped.

To trust a private CA, pass your own client. `WithSecureSSL` has no effect on a client you supply; its own TLS settings decide.

```go
pool := x509.NewCertPool()
pool.AppendCertsFromPEM(caPEM)
client := &http.Client{Transport: &http.Transport{
	TLSClientConfig: &tls.Config{RootCAs: pool},
}}
zr := api.NewZipReport(serverURL, apiKey, processor.WithHTTPClient(client))
```

#### Builds fail on any file error

`report.Build`, `report.BuildZipFs` and `zipreport build` used to skip files they could not process. They now fail, and `BuildZipFs` returns a nil `ZipFs`. These are fatal:

- a file that cannot be read;
- `index.html` or `manifest.json` that is a symlink;
- with `-s` (`FollowSymlinks`): a broken or looping symlink, a symlinked folder, or a symlink that points outside the template folder.

Check `BuildResult` and handle the nil `ZipFs`. Fix the listed files, or remove them from the template folder.

#### `data.json` must be a JSON object

A malformed `data.json`, or one whose top level is not an object (an array, a string, a number), now fails every render with `invalid data.json`. Before, it was ignored.

Still accepted: an empty file, `null`, and a file that starts with a UTF-8 byte-order mark.

#### Folder templates are contained

A template loaded from a folder can no longer read or write outside that folder, whether through `..` or a symlink that resolves outside it.

- `report.LoadDir` fails if `index.html` or `manifest.json` is such a symlink. The error wraps `fileutils.ErrOutsideFolder`.
- Includes, imports and assets that resolve outside the folder are reported as not found.
- `Save()` skips them, so they are not sent to the server.

A dangling symlink inside the folder is rejected the same way.

If your templates share files through symlinks to a common folder outside them, copy the shared files into each template. `zipreport build -s` does not help: a build also rejects symlinks that point outside the template folder.

#### `manifest.json` `report` must be a relative path

The `report` field (the entry file) must be a relative path that stays inside the report. A leading `/` is removed, so `/index.html` still works. Anything that leaves the report, such as `../other.html`, fails to load with an error that wraps `report.ErrInvalidManifest`.

#### Unportable file names fail

A name is unportable if it has a `..` segment (between `/` or `\`), a drive-letter prefix such as `a:`, or starts with `\`. Such names behave differently across operating systems and can escape the report folder.

- **Folders:** `Save()`, renders and builds fail and name the file.
- **`.zpt` files:** loading fails with an error that wraps `fileutils.ErrUnportableName`.
- **In code:** `ZipFs.Add`, `ZipFs.Mkdir`, `OverlayFs.Add` and `ReportFile.Add` return `fileutils.ErrUnportableName` immediately. `Add` with an empty name or `.` returns `fileutils.ErrInvalidPath`.

Rename the files. `fileutils.IsUnportableName(name)` tells you in advance whether a name is affected.

#### Size and count limits

New limits protect against oversized or malicious input. Most reports are far below them.

**Loading a `.zpt` file** fails with `fileutils.ErrZipTooLarge` if it has:
- more than 512 MiB of uncompressed content;
- more than 10,000 entries;
- an entry name longer than 1,024 bytes;
- folders nested deeper than 32 levels.

**Rendering or saving a folder template** fails with `fileutils.ErrZipTooLarge` past:
- 10,000 files and folders (a folder reached through several symlinks counts once per path);
- 100,000 listed directory entries;
- `MaxUncompressedSize` bytes of content;
- 32 levels of nesting.

**PDF responses** larger than 512 MiB fail with `processor.ErrResponseTooLarge`.

To raise the byte limits:

```go
// Process-wide. Set it once at startup, before any report is loaded.
fileutils.MaxUncompressedSize = 2 << 30

zr := api.NewZipReport(serverURL, apiKey, processor.WithMaxResponseSize(1<<30))
```

The entry, name-length and nesting limits are fixed. `api.RenderPDF` takes no options, so its PDF response limit stays at 512 MiB; use `api.NewZipReport` with `WithMaxResponseSize` instead. `MaxUncompressedSize` is process-wide and applies to `api.RenderPDF` too.

#### Renders no longer write into your `ReportFile`

`api.ZipReport` and `api.MIMEReport` render into a private copy of the report (see [Renders work on a copy](#renders-work-on-a-copy-of-the-report)). In 1.0.x the rendered `report.html` and generated `_gen_*` images were left in the `ReportFile` you passed in. Now they are not, so `zpt.GetBytes("report.html")` after a render fails.

Use the `JobResult` instead: `Report` holds the PDF or MIME message. If you need the rendered HTML itself, render with `template.NewMiyaRender` on `zpt.Clone()` and read `report.html` from that clone.

#### A `MiyaRender` renders once

A second `Render`, `RenderTemplate` or `RenderString` on the same `template.MiyaRender` returns `template.ErrRendererUsed`. In 1.0.x, rendering the same template twice returned its raw source; other repeat calls (several `RenderString`s, or a retry after a missing-parameter error) worked.

Create a new renderer for each render, including a retry, and configure it with `GetEnvironment`/`SetWrapper` before rendering. `api.ZipReport` and `api.MIMEReport` already do this, so code that only uses `pkg/api` is unaffected.

```go
out, err := template.NewMiyaRender(zpt.Clone()).Render(data)
// retry: build another renderer
```

#### MIME resource paths are relative to the report file

`MIMEReport` now resolves `src` and `<link href>` references relative to the folder of the manifest's report file, as a browser would. A leading `/` means the report root. In 1.0.x every reference was looked up from the report root as written.

This only matters if the entry file is in a subfolder (for example `"report": "mail/index.html"`) and its references were written relative to the root. Make them relative to the entry file, or start them with `/`.

#### MIME headers reject control characters

`MIMEReport` now returns an error if Subject, From or To contain CR, LF, NUL or another control character, or if an ASCII subject is longer than 989 characters. Before, such values could inject extra headers. If these fields come from user input, strip or reject control characters first.

### Changes you may notice

These don't break code, but output or behaviour differs.

#### Renders work on a copy of the report

Each API render now writes into a private, in-memory copy of the loaded report (`ReportFile.Clone()`). A loaded report is never modified, so one `ReportFile` can be shared across goroutines.

- **Your data map** is not modified, at its top level. Templates can still assign into nested maps and slices you pass in.
- **Custom filters:** a filter that writes into a `ReportFile` it captured when you set it up writes into the original report, not into the copy being rendered. Its files are not sent with that render.
- **Debug server:** renders also use a copy, so `report.html` and generated images are no longer written into the template folder.

#### What a folder template sends to the server

When a folder template is rendered (`Save()`, `BuildFromReportFile`):

- every non-hidden file in the folder is sent on every render;
- root-level `_gen_*` files (images generated by earlier renders) are skipped;
- the `manifest.json` on disk is sent. Changes made through `GetManifest()` are not sent;
- the render fails if there is no `index.html`;
- unreadable files and subfolders (permission errors) are skipped silently. Builds, by contrast, fail on them;
- custom `FsInterface` implementations work. In 1.0.x `BuildFromReportFile` returned `unsupported file system type`. Their `List` methods must return base names.

#### `.zpt` output is deterministic

Building the same folder twice with the same Go toolchain produces byte-identical `.zpt` files.

#### Builds handle more folder shapes

`BuildZipFs` and `zipreport build` now follow a symlinked source folder and accept a source folder whose name starts with `.`. Both used to produce an empty archive. Sockets, FIFOs and devices are skipped.

#### Server backend

- Option order no longer matters; for example, `WithTimeout` before or after `WithHTTPClient` gives the same result.
- `WithHTTPClient(c)` combined with `WithTimeout` applies the timeout to a copy of `c`; your client is not changed.
- `WithHTTPClient(nil)` uses the default client.
- The response content type is checked before the body is read.
- Error bodies from the server are read up to a bound and truncated to 1 KiB in error messages.

#### MIME output

- base64 lines are wrapped at 76 characters;
- non-ASCII subjects are RFC 2047-encoded, and long `To` headers are folded;
- only `src` and `<link href>` files that exist in the report are embedded. `.html` files and hidden files are never embedded;
- `<a href>` links are left untouched;
- line breaks in the title are collapsed;
- the report file named in the manifest is used, and a missing one is reported as `failed to get <file>` rather than `failed to get report.html`;
- Subject, From and To are trimmed, and a blank subject falls back to the report title;
- a file referenced several times is embedded once;
- non-ASCII attachment file names are RFC 2231-encoded;
- references with any URL scheme are left alone, not only `http:`, `data:` and `cid:`.

#### Filter registration errors are returned

If registering the built-in filters fails, the render now returns that error instead of continuing without them.

#### Debug server details

- It sets a 10-second `ReadHeaderTimeout`.
- The startup message shows the bind address, `http://127.0.0.1:8000` by default.
- A request for a dotfile gets the rendered report, not the file.

### Checking errors

Errors from loading (`report.Load`, `LoadFile`, `LoadBytes`, `LoadDir`) wrap the named errors, so use `errors.Is`:

```go
zpt, err := report.LoadFile(path)
switch {
case errors.Is(err, fileutils.ErrZipTooLarge):
	// over a size or count limit
case errors.Is(err, fileutils.ErrUnportableName), errors.Is(err, fileutils.ErrOutsideFolder):
	// a bad file name or path; both also match fileutils.ErrInvalidPath
case errors.Is(err, report.ErrInvalidManifest), errors.Is(err, report.ErrMissingField):
	// manifest.json problem
}
```

Render results (`*report.JobResult`) carry the error as a string in `Error`, so `errors.Is` does not work on them. Check `Success`, and match the message text if you need to tell errors apart.
