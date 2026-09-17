# LocalBridge / Su~ Product Requirements

## 1. Product Goal

LocalBridge / Su~ is a Windows-first local-network transfer application written primarily in Go.

The project originally provides a simple HTTP-server-based text clipboard workflow:

* monitor/read text clipboard content;
* set text clipboard content;
* allow LAN devices to interact with the clipboard through HTTP.

This existing text clipboard capability must remain available.

The project now needs to solve a broader problem:

**reliable transfer of files, images, videos, text files, and text between Windows and iPhone/iPad/other LAN devices.**

iPhone cannot rely on desktop-style file clipboard behavior for arbitrary files, images, and videos.

Therefore, non-text transfer should primarily use:

**Windows desktop application + local HTTP server + QR code + mobile web interface + upload/download APIs.**

The product should remain usable entirely inside a local network.

---

# 2. Main Windows UI

The Windows graphical application must provide at least three primary areas:

1. 文件分享
2. 接收记录
3. 设置

The existing clipboard functionality may remain as an API, background feature, or additional UI feature according to the existing architecture.

Do not remove the clipboard functionality.

If a GUI framework has already been selected and is viable, continue using it rather than replacing it without a strong technical reason.

---

# 3. 文件分享

The 文件分享 page contains three conceptual areas:

## 3.1 QR / Share Access

Display a QR code and usable LAN URL for accessing shared files.

The application must avoid selecting obviously unusable addresses such as:

* `127.0.0.1`;
* loopback-only addresses;
* inappropriate virtual adapters when a reachable LAN address is available.

If multiple usable network interfaces exist, provide a reasonable mechanism to choose the share address.

The UI should expose:

* current share address;
* QR code;
* URL;
* copy URL action.

---

## 3.2 File Selection

Windows users must be able to add files by:

### File picker

Clicking the selection area opens the native or appropriate Windows file picker.

Multiple files can be selected at once.

### Drag and drop

Files can be dragged from Windows Explorer into the application.

Support:

* single file;
* multiple files;
* image;
* video;
* text file;
* common arbitrary files.

---

# 4. Share Session

A single selection/drop operation containing one or more files creates one logical:

**Share Session**

Example:

First operation:

* A.jpg
* B.mp4
* C.txt

creates Share Session A.

Second operation:

* D.pdf
* E.zip

creates Share Session B.

The Windows share list should display sessions rather than flattening every file into unrelated entries.

Each session should be expandable/collapsible.

When expanded, display its files.

Useful per-file information includes:

* filename;
* size;
* type;
* transfer/share status when relevant.

Users must be able to:

* remove an individual share session;
* clear all active share sessions.

If supported cleanly by the current architecture, removing an individual file from a session may also be provided.

When a share or shared file is removed, its public transfer endpoint must no longer grant access to that removed content.

---

# 5. Share Tokens

Externally accessible share URLs must not expose arbitrary local filesystem paths.

Use application-managed IDs/tokens.

Share identifiers exposed to LAN clients should be sufficiently unpredictable to avoid trivial guessing.

A client must only gain access to files explicitly included in the corresponding share.

---

# 6. Mobile Share Page

Scanning a share QR code should open a mobile-friendly web page representing the corresponding Share Session.

Do not make a multi-file QR code point directly to only one raw file.

The page should list every file in the session.

## Images

Support:

* preview;
* open;
* download/save;
* system share when browser capabilities allow it.

## Videos

Support:

* basic browser preview when practical;
* download;
* open;
* system share when available.

## Text Files

Support:

* filename;
* size;
* download;
* optional safe text preview.

Do not automatically load very large text files entirely into the page merely for preview.

## Other Files

At minimum show:

* filename;
* size;
* download/open action.

The page must remain usable even when optional browser sharing APIs are unavailable.

---

# 7. iPhone / iPad Share Integration

Where supported, the mobile web page may use browser capabilities such as:

* `navigator.share()`;
* `navigator.canShare()`.

These capabilities should be treated as progressive enhancement.

They must not be the only way to retrieve a file.

Every supported file must still have a reliable ordinary HTTP download/open path.

The base workflow must continue functioning when the system share API is unavailable.

---

# 8. Shortcut-Friendly HTTP Interface

The HTTP interface should remain simple enough for future iPhone/iPad Shortcuts integration.

A shortcut should be able to:

* access share metadata;
* obtain file information;
* directly download a selected file through HTTP.

Do not make file download depend exclusively on JavaScript UI state.

Actual endpoint naming should follow the final project architecture.

Do not invent duplicate APIs when an equivalent stable API already exists.

---

# 9. 接收记录

The 接收记录 area covers files received by Windows from other LAN devices.

It must provide access to a QR code or URL for the mobile upload page.

The mobile upload page must allow users to select and send:

* images;
* videos;
* text files;
* other files;
* multiple files.

The upload interface should display meaningful:

* progress;
* current state;
* success;
* failure.

Windows should display received transfer history/records.

Useful information includes:

* receive time;
* filename;
* size;
* status;
* resulting save location;
* session grouping when multiple files belong to one operation.

---

# 10. Automatic Receive

Settings must include:

**自动接收文件**

When enabled, accepted uploads may be saved automatically according to application settings.

When disabled, the application must provide a clear Windows-side decision flow such as:

* receive;
* reject.

The HTTP server must not be designed in a way that deadlocks while waiting for GUI interaction.

Transfer state should explicitly model intermediate conditions where needed.

Possible states include concepts such as:

* pending;
* uploading;
* waiting_confirmation;
* accepted;
* rejected;
* completed;
* failed.

Exact names should follow the existing code style.

---

# 11. Resumable Downloads

File downloads must support real HTTP resumability.

The server must correctly handle relevant range behavior, including:

* `Range`;
* `206 Partial Content`;
* `Content-Range`;
* `Accept-Ranges`;
* `Content-Length`;
* `HEAD`;
* invalid/out-of-range requests;
* `416 Range Not Satisfiable` when appropriate.

Where useful, use validators such as:

* ETag;
* Last-Modified;
* If-Range.

Large files must be streamed.

Do not load an entire large file into memory before sending it.

Tests should verify at least:

* full download;
* partial range download;
* range beginning at a non-zero offset;
* interrupted/resumed download behavior;
* invalid ranges;
* large-file streaming behavior.

---

# 12. Resumable Uploads

Uploading must support true resume after interruption.

A single ordinary multipart upload that always restarts from byte zero does not satisfy this requirement.

Use an Upload Session / offset-based design or a compatible existing implementation.

The design should support concepts equivalent to:

1. create upload;
2. obtain upload identifier;
3. query acknowledged offset;
4. send data starting from the acknowledged offset;
5. validate the requested offset;
6. persist partial content;
7. resume after connection failure;
8. validate completion;
9. finalize the file.

A tus-style offset workflow may be used as design inspiration, but implementing the complete tus protocol is not mandatory unless it is the best fit for the existing project.

Partial uploads should use temporary files or equivalent safe storage.

When upload completion is verified, finalize/move the file safely.

Offset disagreement must not silently corrupt or overwrite transferred data.

The client should be able to query the server and continue from the server-confirmed position.

---

# 13. Transfer Integrity and Filesystem Safety

File operations must handle:

* filename sanitization;
* path traversal;
* duplicate filenames;
* temporary files;
* interrupted transfers;
* finalization;
* large files;
* application restart behavior where practical.

Client-provided filenames must never be treated as trusted filesystem paths.

Inputs such as:

`../../example`

must never escape the configured receive directory.

Do not expose arbitrary Windows absolute paths through public APIs.

---

# 14. Settings

The 设置 page must include the following categories.

## 14.1 Software

### 开机自启

Allow users to enable and disable Windows startup behavior.

The setting must reflect actual Windows integration, not merely store a Boolean in configuration.

The application should be able to determine or reconcile the real startup state.

Prefer a solution that does not require administrator privileges when possible.

### 关闭按钮最小化到托盘

When enabled:

closing the main window minimizes/hides the application to the system tray and the HTTP server continues running.

When disabled:

normal close behavior should exit the application.

### Windows Right-Click Menu

Allow enabling/disabling Explorer integration.

When enabled, selecting one or multiple files and using the Windows context menu should provide:

**通过 Su~ 分享**

When the application is already running:

send the selected files to the existing application instance and create a Share Session.

When the application is not running:

start the application and create the Share Session.

The design must avoid launching multiple independent HTTP server instances.

Prefer a single-instance mechanism with IPC, Named Pipe, local socket, or another appropriate Windows mechanism.

Must correctly handle:

* one file;
* multiple files;
* paths with spaces;
* Unicode;
* Chinese filenames/paths;
* executable paths containing spaces.

Disabling the setting must remove/revert the corresponding Windows shell integration.

### 还原所有设置

Provide an action to restore application settings.

Restoring must reconcile external Windows integration where applicable, including:

* startup registration;
* Explorer context menu.

Ask for confirmation before performing a broad reset.

---

# 15. System Tray

The Windows application should provide a usable tray presence where supported by the selected GUI architecture.

At minimum:

* open/show main window;
* expose running status where practical;
* exit application.

Explicit Exit must correctly shut down:

* HTTP server;
* background goroutines;
* watchers;
* IPC resources;
* temporary application resources that require cleanup.

---

# 16. Notifications

Settings must include notification controls.

At minimum:

* notification sound enabled/disabled;
* receive-complete sound;
* send/download-complete sound where completion can be determined reliably.

A default built-in/system sound is sufficient for the baseline requirement.

Custom sound-file selection is optional and must not delay the core product.

---

# 17. Configuration Persistence

User settings must persist across application restarts.

Relevant settings may include:

* server/listen port;
* selected LAN address/interface;
* receive directory;
* automatic receive;
* close-to-tray;
* startup behavior;
* Explorer integration;
* notification settings.

Configuration must:

* provide sane defaults;
* handle first run;
* handle missing configuration;
* handle malformed configuration safely;
* support future schema evolution reasonably.

Do not hard-code developer-machine-specific paths.

---

# 18. Core Domain Concepts

The implementation should have clear ownership for concepts equivalent to:

* Share Session;
* Shared File;
* Receive / Transfer Session;
* Received File;
* Upload Session;
* Application Settings.

Exact type and package names are not prescribed.

Reuse existing project concepts when they already model these responsibilities correctly.

Avoid having GUI components, HTTP handlers, and Windows integration independently maintain conflicting versions of the same application state.

---

# 19. Architecture Direction

The project remains primarily a Go application unless the existing repository already contains justified supporting technology.

Conceptually separate responsibilities such as:

* desktop UI;
* application/service layer;
* HTTP API;
* file transfer;
* storage/state;
* configuration;
* Windows integration;
* clipboard.

These names do not mandate specific folders.

Do not split code merely to create more packages.

The goal is understandable responsibility boundaries.

Avoid one giant entry-point file containing unrelated logic for:

* HTTP;
* GUI;
* QR;
* transfer;
* registry;
* configuration;
* clipboard;
* storage.

Perform only necessary refactoring.

---

# 20. Security

Although the application is intended for a LAN, do not assume every LAN client is trusted.

At minimum protect against:

* path traversal;
* arbitrary filesystem reads;
* arbitrary filesystem overwrite;
* unsafe filenames;
* trivially guessable share access;
* uncontrolled upload sizes;
* accidental disk exhaustion where reasonable;
* invalid transfer offsets.

Security mechanisms should remain proportional to a local transfer application.

Do not introduce unnecessary enterprise authentication complexity unless required by an existing architecture or future explicit requirement.

---

# 21. Required End-to-End Scenarios

Before final completion, verify representative real workflows.

## Windows → iPhone

Verify:

* share an image;
* scan QR;
* open mobile page;
* preview/open image;
* download/save image.

Verify:

* video;
* PDF;
* text file;
* arbitrary file;
* multiple files in one Share Session.

## iPhone → Windows

Verify:

* scan upload QR;
* select image;
* upload;
* Windows receives it.

Also verify:

* ordinary file;
* multiple files.

## Resumability

Using a sufficiently large file:

* start download;
* interrupt;
* resume without retransferring the entire file from byte zero where the client supports resume.

Separately:

* start upload;
* interrupt;
* query/recover current offset;
* resume from acknowledged offset;
* complete correctly.

## Explorer Integration

Verify:

* one selected file;
* multiple selected files;
* Chinese path;
* Unicode filename;
* filename/path containing spaces.

---

# 22. Build / Run / Release Product Expectations

The completed repository must expose clear workflows for:

* development run;
* build;
* test;
* release.

Reuse/consolidate existing scripts where possible instead of creating many equivalent entry points.

A release build must not depend on:

* IDE-only state;
* user-specific absolute paths;
* untracked development artifacts.

Release output should be minimal and understandable.

---

# 23. Final Documentation Expectations

After the implementation is complete, consolidate project documentation.

The final maintained documentation should allow a developer to quickly understand the application without reading historical AI planning files.

A useful final structure may include:

## README.md

Explain:

* what the product is;
* primary capabilities;
* basic usage;
* development entry points;
* links to deeper documentation.

## docs/ARCHITECTURE.md

Explain concisely:

* overall architecture;
* process/runtime structure;
* GUI ↔ application ↔ HTTP relationships;
* Windows → mobile data flow;
* mobile → Windows data flow;
* configuration;
* storage;
* Windows integration;
* major directory structure.

Prefer diagrams, directory trees, tables, and concise descriptions over essay-style prose.

## docs/MODULES.md

For important modules/packages explain:

* responsibility;
* important types/interfaces;
* dependencies;
* where to modify common features.

The goal is fast code navigation.

## docs/API.md

Document APIs that actually exist in the final implementation.

Include as appropriate:

* method;
* path;
* purpose;
* parameters;
* response;
* status codes;
* range behavior;
* resumable upload behavior.

Do not document APIs that are only planned.

## docs/DEVELOPMENT.md

When useful, document:

* dependencies;
* development setup;
* Run;
* Build;
* Test;
* Release;
* Windows-specific development notes.

If this content is small, merge it into README instead of creating another document.

---

# 24. Final Product Acceptance

The product is considered complete only when the final implementation satisfies the applicable requirements above and, at minimum:

1. existing text clipboard functionality still works;
2. Windows provides a usable GUI;
3. files can be selected;
4. multi-select works;
5. drag-and-drop works;
6. each selection batch creates a Share Session;
7. session contents can be inspected;
8. active shares can be removed;
9. all shares can be cleared;
10. share QR codes work from an iPhone on the LAN;
11. the mobile share page lists session files;
12. supported media can be previewed/opened appropriately;
13. files can be downloaded normally;
14. direct HTTP file download is available;
15. mobile users can upload files;
16. Windows shows receive records;
17. automatic receive behavior works according to settings;
18. downloads support real HTTP Range resumability;
19. uploads support true offset-based resume;
20. tray behavior works;
21. startup integration works;
22. `通过 Su~ 分享` Explorer integration works;
23. notification settings work;
24. settings persist;
25. important transfer/security behavior has meaningful tests;
26. Build succeeds;
27. Run workflow succeeds;
28. Release workflow succeeds;
29. release output is clean;
30. obsolete documentation has been removed or merged;
31. maintained architecture/module/API documentation matches the final code.

Implementation details may evolve based on the existing repository.

The priority is a reliable, maintainable working application rather than mechanically following an imagined architecture.
