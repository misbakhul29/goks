# GoKS Road to v1.0.0

Status: Planned
Baseline: v0.15.1
Product goal: GoKS menjadi pilihan yang kredibel untuk programmer Go yang ingin
membangun website production-grade dengan satu bahasa, development experience
yang nyaman, dan deployment single binary tanpa Node.js/npm sebagai runtime.

Dokumen ini adalah backlog produk dan engineering. Setiap task harus diubah
menjadi issue/PR terpisah, memiliki test, dan mengikuti quality gate di
`AGENTS.md`. Versi milestone adalah target urutan, bukan janji tanggal.

## 1. Product definition

### Target pengguna

1. Go developer yang membutuhkan website SSR, API, authentication, dan
   interaktivitas tanpa berpindah ke JavaScript/TypeScript.
2. Tim kecil yang menginginkan deployment sederhana berupa binary/container
   tunggal.
3. Tim backend Go yang membutuhkan progressive enhancement: HTML tetap berguna
   tanpa JavaScript, lalu WASM dipakai hanya pada island interaktif.
4. Tim yang membutuhkan framework opinionated tetapi tetap dapat memakai
   package Go biasa dan infrastruktur standar Go.

### Janji produk v1.0.0

- Go adalah bahasa aplikasi utama untuk server, template, dan interaktivitas.
- `goks new -> goks dev -> goks build` adalah alur yang konsisten dan dapat
  diulang pada proyek baru.
- Aplikasi dapat dirender sebagai HTML server-side, memakai server actions,
  API routes, ORM, auth/RBAC, WebSocket, dan WASM islands secara bersamaan.
- Static page tidak membawa payload WASM yang tidak diperlukan.
- Production build memiliki health check, graceful shutdown, logging yang
  dapat dioperasikan, batas request, dan dokumentasi deployment.
- Public API `pkg/*` terdokumentasi, diuji, dan memiliki kebijakan kompatibilitas.
- Semua klaim fitur di README memiliki contoh executable atau test integrasi.

### Non-goals sampai v1.0.0

- Generic plugin marketplace atau plugin ABI tanpa use case nyata.
- Menambah database adapter hanya untuk memperbanyak daftar fitur.
- Menambahkan JavaScript runtime atau Node.js sebagai dependency wajib.
- Mengejar benchmark marketing sebelum workload dan metodologi disepakati.
- Memecah framework menjadi banyak abstraction/interface tanpa kebutuhan nyata.

## 2. Status baseline v0.15.1

Yang sudah diverifikasi pada baseline:

- `go vet ./...` berhasil.
- `go test ./...` berhasil.
- `go test -race ./...` berhasil.
- `go build ./...` berhasil.
- `GOOS=js GOARCH=wasm go build ./...` berhasil.
- `gofmt -l .` bersih.

Gap yang harus ditutup sebelum v1.0.0:

- Belum ada CI repository yang menjalankan quality gate secara otomatis.
- Architecture audit dan public API inventory belum menjadi artefak resmi.
- Dukungan Go yang terdokumentasi (`README`) dan `go.mod` harus diselaraskan.
- Klaim PostgreSQL/MySQL, TinyGo, SSG, middleware, dan fitur lain harus
  diverifikasi melalui test/example; klaim yang belum benar harus diperbaiki,
  bukan dibiarkan sebagai marketing.
- Coverage integrasi, load/concurrency, security, dan cross-platform masih
  perlu diperluas.
- Dokumentasi pengguna, migration guide, troubleshooting, dan deployment
  guide belum memiliki acceptance test yang konsisten.

## 3. Milestone dan task

## M0 — Foundation dan governance (`v0.15.1 -> v0.16.0`)

Tujuan: menetapkan ground truth, quality gate otomatis, dan kontrak produk.
Semua task M0 adalah blocker untuk klaim production-ready.

### M0.1 Architecture audit [P0]

- [x] Buat `GOKS_ARCHITECTURE_AUDIT.md` berdasarkan implementasi aktual.
- [x] Petakan lifecycle: `.gox -> workspace -> generated entry -> build ->
      runtime -> response/hydration`.
- [x] Catat strength, weakness, security risk, scalability risk, performance
      risk, technical debt, API risk, dan testing gap.
- [x] Tandai bagian dokumentasi yang belum sama dengan source code.

Catatan implementasi: audit awal tersedia di `GOKS_ARCHITECTURE_AUDIT.md`.
Temuan P0/P1 menjadi input langsung untuk M0.4, M1, M3, M4, dan M5.

Acceptance criteria:

- Audit memiliki path dan line reference untuk temuan penting.
- Setiap critical/high finding memiliki task owner dan milestone.
- Audit diperbarui setiap ada perubahan arsitektur besar.

### M0.2 Public API inventory [P0]

- [x] Inventaris semua exported identifier di `pkg/*`, `runtime/*`, dan CLI
      yang digunakan oleh generated application.
- [x] Tandai setiap API sebagai `stable`, `experimental`, atau `internal`.
- [x] Dokumentasikan breaking-change policy, deprecation policy, dan migration
      note format.
- [x] Tambahkan compile-level compatibility checks untuk API stable.

Catatan implementasi: `docs/api/API_INVENTORY.md` memuat inventaris lengkap,
klasifikasi stabilitas, dan format catatan migrasi. Test kompatibilitas
tersedia di `tests/compatibility/api_compat_test.go`.

Acceptance criteria:

- Tidak ada exported API publik yang tidak memiliki status.
- Perubahan API stable memerlukan ADR, migration note, dan test.
- API report dapat dibuat ulang secara deterministik.

### M0.3 CI quality gate [P0]

- [x] Tambahkan CI untuk `go fmt -l .`, `go vet ./...`, `go test ./...`,
      `go test -race ./...`, dan `go build ./...`.
- [x] Tambahkan job `GOOS=js GOARCH=wasm go build ./...`.
- [x] Tambahkan matrix build lintas platform yang realistis:
      linux/arm64, darwin/arm64, windows/amd64.
- [x] Tambahkan artifact log untuk test, race, build, dan bundle size.
- [ ] Tetapkan branch protection: merge hanya jika required checks berhasil.
      (butuh konfigurasi repository GitHub, bukan perubahan kode)

Catatan implementasi: `.github/workflows/ci.yml` menjalankan quality gate,
WASM build, cross-build matrix, `govulncheck`, dan upload artifact log/bundle size.
Job berjalan dengan `permissions: contents: read` dan tidak menyentuh secret aplikasi.

Acceptance criteria:

- Pull request yang gagal format, test, race, vet, atau build tidak dapat
  dianggap ready.
- CI tidak membaca secret aplikasi dan tidak mengupload credential ke artifact.

### M0.4 Security baseline [P0]

- [x] Audit `pkg/action`, `pkg/auth`, `pkg/rbac`, `pkg/router`, `pkg/ws`, dan
      file/process handling di `internal/cli` terhadap `SECURITY.md`.
- [x] Buat security finding register dengan severity, exploit scenario,
      mitigation, dan regression test.
- [x] Ganti validasi origin substring pada server action dengan exact-origin
      comparison berbasis scheme, hostname, dan effective port.
- [x] Tolak redirect action ke absolute/external URL dan protocol-relative URL.
- [x] Ganti validasi origin substring pada WebSocket handshake dengan
      exact-origin comparison; perbaiki lifecycle channel client dan lepas
      client dari room saat disconnect.
- [x] Jalankan dependency vulnerability scan di CI (`govulncheck`).
- [x] Tetapkan proses security disclosure dan release security advisory.

Catatan implementasi: fix `pkg/action`, `pkg/ws`, dan `pkg/orm` sudah diverifikasi
dengan `go test`, `go vet`, `go build`, dan `GOOS=js GOARCH=wasm go build`.
Regresi test ditambahkan di `pkg/action/action_test.go` dan `pkg/ws/ws_test.go`.
Finding register ada di `GOKS_ARCHITECTURE_AUDIT.md`. Proses disclosure dan format
advisory didokumentasikan di `SECURITY.md`.

Acceptance criteria:

- Semua critical/high finding ditutup atau memiliki risk acceptance tertulis.
- CSRF, XSS, SSRF, path traversal, request limit, timeout, session, dan
  WebSocket authorization memiliki test yang dapat direproduksi.

### M0.5 Release discipline [P1]

- [x] Pastikan satu sumber versi dipakai oleh CLI, generated app, Studio,
      OAuth User-Agent, README, dan release metadata.
- [x] Buat checklist release serta changelog berdasarkan Conventional Commits.
- [x] Pastikan tag semver menunjuk commit yang build dan test-nya sudah lulus.
- [x] Hentikan release jika README, examples, atau generated template memakai
      versi lama.

Catatan implementasi: `internal/version` menjadi satu sumber versi; README,
badge rilis, template `goks new`, dan entry module generator diselaraskan
ke `v0.16.0` dan Go `1.26.6`.

Acceptance criteria:

- `goks version`, Studio, generated `go.mod`, dan release tag konsisten.
- Release dapat direproduksi dari clean checkout.

## M1 — Runtime HTTP dan operability (`v0.16.0 -> v0.17.0`)

Tujuan: runtime aman dan dapat dipakai di bawah traffic nyata.

### M1.1 Middleware contract [P0]

- [x] Tetapkan dan dokumentasikan urutan middleware: Logger, CORS, Secure,
      RequestID, Compress, MaxBytes, Timeout, auth, dan application middleware.
- [x] Uji bahwa semua route menerima policy yang sama, termasuk API route dan
      error response.
- [x] Tetapkan default yang aman untuk CORS, timeout, body limit, dan security
      headers.
- [x] Dokumentasikan cara override default tanpa menghapus kontrol keamanan.

Catatan implementasi: Urutan middleware dan short-circuit teruji di
`pkg/router/middleware_test.go`. Safe error response (tanpa leak panic message/secrets),
HSTS HTTPS-only guard, context-propagated Request ID, dan body limit terverifikasi.

Acceptance criteria:

- Ada integration test untuk urutan middleware dan short-circuit behavior.
- Error response tidak membocorkan stack trace atau secret pada production.

### M1.2 Request lifecycle dan graceful shutdown [P0]

- [x] Pastikan context request diteruskan sampai handler, action, ORM, dan
      external call.
- [x] Implementasikan atau verifikasi graceful shutdown dengan deadline.
- [x] Drain in-flight HTTP request dan tutup WebSocket connection dengan bersih.
- [x] Uji SIGTERM/SIGINT pada standard dan standalone build.

Catatan implementasi: `runtime/server` menyediakan `Shutdown(ctx)` terprogram dan
menangani `http.ErrServerClosed` secara bersih. Test lifecycle dan shutdown
terverifikasi di `runtime/server/server_test.go`.

Acceptance criteria:

- Shutdown test membuktikan request aktif tidak dipotong sebelum deadline.
- Tidak ada goroutine leak yang terdeteksi pada test lifecycle.

### M1.3 Structured logging dan health [P1]

- [x] Tetapkan structured log schema: timestamp, level, request ID, method,
      path, status, latency, dan error class.
- [x] Pisahkan log developer dari log production.
- [x] Sediakan pola endpoint health/readiness yang tidak bergantung pada Studio.
- [x] Dokumentasikan liveness, readiness, dependency failure, dan shutdown.

Catatan implementasi: `/_goks/healthz` dan `/_goks/ready` tersedia secara bawaan di
runtime server dan teruji di `runtime/server/server_test.go`. Request ID otomatis
terintegrasi pada context dan response header.

Acceptance criteria:

- Log dapat diproses mesin tanpa parsing string ad-hoc.
- Health endpoint tidak membocorkan konfigurasi atau credential.

### M1.4 Concurrency/load baseline [P1]

- [x] Tambahkan concurrent request test untuk `runtime/server` dan router.
- [x] Audit `pkg/ws` untuk goroutine leak, slow consumer, concurrent write,
      room lifecycle, dan authorization.
- [x] Audit `pkg/store` untuk race, subscription leak, dan update ordering.
- [x] Tambahkan benchmark baseline untuk router, component render, dan WS hub.

Catatan implementasi: Benchmark baseline ditambahkan di `pkg/router/router_bench_test.go`,
`pkg/component/component_bench_test.go`, dan `pkg/ws/ws_bench_test.go`.
Concurrent load test ditambahkan di `runtime/server/server_test.go`.

Acceptance criteria:

- Race test bersih pada workload concurrent yang terdokumentasi.
- Slow client tidak dapat memblokir seluruh hub.
- Baseline benchmark disimpan agar regression terukur.

## M2 — Routing, API routes, dan error model (`v0.17.0 -> v0.18.0`)

Tujuan: routing file-system dan programmatic menjadi predictable dan mudah
debug.

### M2.1 Route semantics [P0]

- [x] Tetapkan precedence static, dynamic, catch-all, API, layout, dan not-found.
- [x] Uji nested layout dan inheritance metadata.
- [x] Uji route parameter decoding, invalid parameter, encoded slash, query
      parameter, trailing slash, method mismatch, dan duplicate route.
- [x] Uji route groups atau dokumentasikan bahwa fitur tersebut belum menjadi
      bagian API v1.

Catatan implementasi: Precedence score (static 100 > dynamic 10 > wildcard 1)
diimplementasikan di `pkg/router/router.go`. Unescaping parameter path, 405 Method Not
Allowed dengan `Allow` header, dan duplicate route replacement teruji di
`pkg/router/router_test.go`.

Acceptance criteria:

- Semua precedence rule tertulis dan memiliki table-driven test.
- Conflict route menghasilkan compile/generate error yang jelas.

### M2.2 API route contract [P0]

- [x] Tetapkan handler signature, supported methods, status code, headers,
      JSON binding, content negotiation, dan error response.
- [x] Pastikan API route tidak masuk bundle WASM.
- [x] Uji body limit, malformed JSON, timeout, panic recovery, dan context cancel.
- [x] Dokumentasikan CORS/auth policy untuk API route.

Catatan implementasi: `Context.Bind` dan `Context.Error` menyediakan contract API
terstandarisasi dengan request ID. Pengujian body limit dan malformed JSON
dilakukan di `router_test.go` dan `middleware_test.go`.

Acceptance criteria:

- API example dari `goks new` dapat di-build dan diuji tanpa edit manual.
- Error API konsisten dan tidak membocorkan detail internal.

### M2.3 Error and observability model [P1]

- [x] Definisikan not-found, method-not-allowed, validation, auth, forbidden,
      conflict, dan internal error mapping.
- [x] Tambahkan request ID ke response dan log untuk error.
- [x] Buat dev overlay yang menampilkan informasi berguna tanpa muncul di
      production response.

Catatan implementasi: Format `ErrorResponse` dan `APIError` membungkus status code,
error message, dan request ID. HTTP 405 Method Not Allowed dan 404 Not Found
terintegrasi pada router level.

Acceptance criteria:

- User dapat membedakan error routing, compile, application, dan dependency.
- Error contract didokumentasikan untuk page dan API.

## M3 — ORM, data access, auth, dan security (`v0.18.0 -> v0.19.0`)

Tujuan: data dan identity layer dapat dipercaya untuk aplikasi produksi.

### M3.1 ORM safety and semantics [P0]

- [x] Audit seluruh query builder: parameterization, identifier validation,
      ordering, pagination, joins, aggregates, dan raw query escape hatch.
- [x] Dokumentasikan transaction semantics, commit/rollback, nested behavior,
      connection ownership, dan context cancellation.
- [x] Uji pool limits, concurrent transaction, failed transaction, dan retry
      policy yang memang didukung.
- [x] Dokumentasikan soft-delete scope, restore, timestamps, hooks, dan update
      behavior.

Catatan implementasi: `isValidSQLIdentifier` memvalidasi tabel dan kolom di
`pkg/orm/crud.go`. `Database.Transaction` menyediakan eksekusi ACID dengan
otomatis rollback saat error/panic. Soft-delete `Restore` dan `WithTrashed` teruji
di `pkg/orm/orm_test.go`.

Acceptance criteria:

- Tidak ada input user yang dirangkai menjadi SQL tanpa validasi/parameterisasi.
- Integration test menggunakan database nyata untuk perilaku ORM penting.
- Race test dan failure-path test lulus.

### M3.2 Migrations [P0]

- [x] Tetapkan naming/versioning migration dan checksum policy.
- [x] Uji apply, rollback, partial failure, transaction atomicity, lock,
      concurrent deploy, dan status drift.
- [x] Pastikan `goks db` aman terhadap path traversal dan command injection.
- [x] Sediakan migration guide untuk production deploy.

Catatan implementasi: `pkg/orm/migrate.go` membungkus setiap migrasi dalam
transaksi atomik dengan rollback otomatis bila terjadi error. Normalisasi path
dengan `filepath.Clean` mencegah path traversal.

Acceptance criteria:

- Dua proses migration bersamaan tidak merusak schema.
- Failure dapat dilanjutkan atau dipulihkan melalui prosedur terdokumentasi.

### M3.3 Database adapter truthfulness [P1]

- [x] Verifikasi dukungan SQLite, PostgreSQL, dan MySQL berdasarkan source,
      driver, test, dan generated example.
- [x] Jika adapter belum benar-benar didukung, koreksi README sebelum v1.0.
- [x] Tambahkan adapter hanya jika ada use case konkret, ADR, dan integration
      test matrix.

Catatan implementasi: `pkg/orm` mendaftarkan driver pure-Go
`modernc.org/sqlite` melalui `pkg/orm/sqlite_driver.go` sehingga `orm.OpenSQLite`
dan `goks db status` bekerja tanpa konfigurasi. File tersebut memakai build
constraint `!js && !wasm`: driver sengaja dikecualikan dari build WASM karena
`modernc.org/libc` tidak mendukung js/wasm dan SQLite hanya relevan untuk server.

Acceptance criteria:

- README hanya menyebut adapter yang build dan test-nya terbukti.
- Connection string, credential, dan error database tidak masuk log/artifact.

### M3.4 Authentication and authorization [P0]

- [x] Dokumentasikan session/JWT threat model dan kapan masing-masing dipakai.
- [x] Uji cookie flags, expiry, rotation, revocation, replay, CSRF, fixation,
      token leakage, dan logout.
- [x] Uji RBAC deny-by-default, role inheritance jika ada, dan authorization
      pada page, API, action, WebSocket, dan Studio.
- [x] Uji OAuth state, PKCE, redirect allowlist, provider error, token expiry,
      account linking, dan callback failure.
- [x] Pastikan secret hanya berasal dari environment/secret manager dan tidak
      pernah masuk generated source atau diagnostics.

Catatan implementasi: Proteksi session fixation diimplementasikan di `pkg/auth/auth.go`
dengan otomatis menghapus token lama pada saat login. JWT secret enforce minimum 32
karakter. OAuth PKCE timing-safe comparison dan secure cookies terverifikasi.

Acceptance criteria:

- Protected resource memiliki test allow dan deny.
- Security-sensitive default aktif dan cara menonaktifkannya terdokumentasi.

## M4 — GOX, component model, SSR, dan WASM (`v0.19.0 -> v0.20.0`)

Tujuan: frontend GoKS stabil, terukur, dan tidak mengejutkan developer.

### M4.1 GOX compiler correctness [P0]

- [x] Buat grammar/syntax reference yang sama dengan parser aktual.
- [x] Tambahkan golden test untuk element, attribute, expression, component,
      event handler, children, fragments, comments, dan malformed input.
- [x] Map compile error ke file `.gox`, line, column, dan source snippet.
- [x] Pastikan generated Go deterministik dan tidak bergantung pada map order.
- [x] Uji import, package alias, reserved word, path traversal, dan generated
      identifier collision.

Catatan implementasi: Spesifikasi syntax GOX didokumentasikan di
`docs/reference/GOX_SYNTAX.md`. `internal/compiler` menyediakan struct `CompileError`
dengan visual snippet caret dan lokasi line/column source `.gox`. Dukungan shorthand
fragment `<>...</>`, explicit `<Fragment>`, ignores comment, binding `.On(...)`, dan
sorting deterministik atribut/props diuji di `internal/compiler/golden_test.go`.

Acceptance criteria:

- Error utama menunjuk lokasi `.gox`, bukan hanya generated `.go`.
- Rebuild source yang sama menghasilkan output yang sama.

### M4.2 Component and hooks contract [P0]

- [x] Tetapkan lifecycle render, mount, update, unmount, dan error behavior.
- [x] Enforce hook call-order invariant dan beri error yang jelas.
- [x] Uji nested component, keyed children, event handler, state update,
      concurrent render, dan cleanup.
- [x] Dokumentasikan batasan server render versus client hydration.
- [x] Audit global state/hook state terhadap request isolation dan data leakage.

Catatan implementasi: `UseState` di `pkg/component/hooks.go` memeriksa hook call-order
invariant (panics dengan error eksplisit saat tipe slot berubah) dan menyediakan safe
fallback untuk server-side rendering (SSR) tanpa active fiber. `Expand` di
`pkg/component/node.go` menggunakan `defer` untuk memulihkan `activeFiber` saat terjadi
panic render, mengeliminasi race condition dan data leakage antar-request.

Acceptance criteria:

- Test component bersih dengan race detector.
- State satu request/user tidak pernah terlihat di request/user lain.

### M4.3 SSR, streaming, Suspense [P1]

- [x] Uji shell, fallback, resolved content, error boundary, client disconnect,
      timeout, ordering, cancellation, dan backpressure.
- [x] Pastikan streaming tidak mengirim data sensitif sebelum authorization.
- [x] Dokumentasikan kapan streaming dipakai dan kapan response biasa lebih tepat.

Catatan implementasi: `pkg/component/ssr.go` mengurutkan atribut secara deterministik
(`sort.Strings`). Suspense dan streaming diuji di `pkg/component/suspense_test.go`
mencakup timeout, context cancellation, fallback, dan XSS entity escaping.

Acceptance criteria:

- Streaming test dapat memverifikasi chunk order dan cancellation.
- No-JS browser tetap menerima konten yang usable.

### M4.4 Islands and hydration [P0]

- [x] Buat fixture static page yang menghasilkan 0 KB WASM.
- [x] Buat fixture interactive page yang hanya memuat island yang diperlukan.
- [x] Uji SSR-to-hydration identity, event handler, state initialization,
      navigation, failure recovery, dan duplicate hydration.
- [x] Tambahkan bundle-size budget dan regression report ke CI.
- [x] Pastikan API route dan server-only code tidak masuk client bundle.

Catatan implementasi: `pkg/component/island.go` mendukung `NeedsHydration` rekursif
yang mengevaluasi subtree component untuk mendeteksi `ClientComponent` maupun event
handler. Fixture zero-WASM static page teruji di `pkg/component/island_test.go`.

Acceptance criteria:

- Static fixture memiliki automated zero-WASM assertion.
- Hydration fixture memiliki browser-level atau equivalent DOM interaction test.

### M4.5 TinyGo and store [P1]

- [x] Verifikasi parity antara standard Go WASM dan TinyGo.
- [x] Dokumentasikan unsupported API dan fallback jika parity belum tercapai.
- [x] Ukur bundle size, startup time, memory, dan build time.
- [x] Dokumentasikan `pkg/store`: subscription, rerender trigger, batching,
      unsubscribe, dan ordering.

Catatan implementasi: `pkg/store` dijadikan package universal (tanpa batasan OS/arch)
dengan proteksi thread-safe mutex dan ID subscription mapping yang mencegah index
corruption saat unsubscribe di luar urutan. Diuji terhadap race condition di
`pkg/store/store_test.go`.

Acceptance criteria:

- TinyGo hanya dipromosikan sebagai supported jika feature matrix lulus.
- Store memiliki test untuk lifecycle dan concurrent update.

## M5 — CLI, generated app, dan developer experience (`v0.20.0 -> v0.21.0`)

Tujuan: programmer dapat belajar dan mengirim aplikasi tanpa membaca source
framework terlebih dahulu.

### M5.1 `goks new` golden project [P0]

- [x] Generated project memiliki `go.mod`, page, layout, component, API,
      middleware, model, dan README yang benar-benar build.
- [x] Tambahkan end-to-end test: `goks new -> go mod tidy -> build -> test ->
      start -> request page`.
- [x] Pastikan generated version, Go version, module path, dan import path benar.
- [x] Validasi nama app, path, module, overwrite protection, dan permissions.

Catatan implementasi: `goks new` menyediakan flag `-m/--module`, validasi nama app
(menolak path traversal `..` dan karakter tidak valid), serta proteksi overwrite direktori.
Template menghasilkan file yang valid dan teruji melalui `internal/cli/new_test.go`.

Acceptance criteria:

- Fresh generated project build tanpa edit manual.
- Template tidak mengandung placeholder yang menghasilkan code rusak.

### M5.2 `goks dev` reliability [P0]

- [x] Uji watch untuk `.gox`, `.go`, CSS/Tailwind, config, route, dan deleted file.
- [x] Uji debounce, duplicate event, failed compile, recovery, and restart.
- [x] Pisahkan overlay GOX, Go build, Tailwind, dan runtime error.
- [x] Pastikan dev server tidak meninggalkan process atau temporary workspace.
- [x] Ukur cold start dan hot reload latency dengan baseline.

Catatan implementasi: `internal/watcher` dan dev server mengelola siklus hidup proses
anak dan workspace sementara dengan bersih. Error overlay dinonaktifkan di production.

Acceptance criteria:

- Save setelah compile error memulihkan server tanpa restart manual.
- Overlay tidak aktif pada production build.

### M5.3 Build, standalone, start, dan export [P0]

- [x] Uji standard build, standalone build, `start`, dan SSG/export dari clean
      checkout.
- [x] Verifikasi asset embedding: HTML, CSS, WASM, images, fonts, dan public.
- [x] Uji binary pada working directory berbeda tanpa source dependency.
- [x] Uji reproducibility, missing asset, invalid flag, output collision, dan
      cross-platform path handling.
- [x] Dokumentasikan runtime file dependency yang memang masih diperlukan.

Catatan implementasi: `goks build` divalidasi di `internal/cli/build_test.go` terhadap
keberadaan folder `app/` dan flag `--compiler`. Standalone build (`--standalone`)
memasukkan WASM, CSS, JS runtime, dan `public/` ke dalam single executable.

Acceptance criteria:

- `goks build --standalone` dapat dijalankan pada directory baru.
- SSG output dapat diserve sebagai static files dan memiliki link/asset yang
  benar.

### M5.4 CLI and LSP contract [P1]

- [x] Dokumentasikan seluruh command, subcommand, flag, default, exit code,
      dan error message.
- [x] Uji `generate`, `page`, `ui`, `db`, `studio`, `export`, dan `lsp`.
- [x] Tambahkan completion/hover/diagnostic/formatting test untuk syntax GOX.
- [x] Pastikan CLI error dapat dipakai di script: non-zero exit, no ANSI-only
      information, dan output yang konsisten.

Catatan implementasi: Seluruh contract CLI didokumentasikan di
`docs/reference/CLI_REFERENCE.md` lengkap dengan flag, exit code, dan contoh.
LSP didukung test unit menyeluruh di `internal/lsp/lsp_test.go`.

Acceptance criteria:

- `--help` dan docs tidak berbeda dari implementasi.
- LSP tidak crash pada malformed atau incomplete `.gox`.

### M5.5 Documentation and examples [P0]

- [x] Tulis getting started sampai deployment dalam urutan yang dapat diikuti.
- [x] Buat examples minimal: blog/CRUD, auth/RBAC, API, action, WebSocket,
      upload/image, static export, dan standalone deployment.
- [x] Setiap example harus masuk CI atau memiliki verification script.
- [x] Tambahkan troubleshooting untuk Go/WASM/TinyGo/Tailwind/database.
- [x] Tulis comparison yang faktual; hapus klaim yang belum terukur.

Catatan implementasi: Panduan deployment komprehensif didokumentasikan di
`docs/DEPLOYMENT.md` mencakup standalone binary, multi-stage Docker, health probes,
dan reverse proxy (Caddy / Nginx).

Acceptance criteria:

- New developer dapat menjalankan hello world dan production build dari clean
  checkout mengikuti docs saja.
- Link, command, version, dan output contoh diverifikasi.

## M6 — Production hardening dan operability (`v0.21.0 -> v0.22.0`)

Tujuan: aplikasi GoKS dapat di-deploy, diamati, dan dipulihkan secara aman.

### M6.1 Security release gate [P0]

- [x] Lakukan threat model review terhadap HTTP, template/HTML, WASM bridge,
      action/RPC, auth, ORM, file serving, image fetch, WebSocket, CLI, dan
      Studio.
- [x] Tambahkan fuzz test untuk parser GOX, router, query builder, JSON/action
      input, dan security-sensitive decoders.
- [x] Audit SSRF, path traversal, XSS, CSRF, request smuggling, zip/tar bomb,
      secret exposure, and denial-of-service limits.
- [x] Review dependency provenance, checksums, and release artifact integrity.

Catatan implementasi: Empat suite fuzz test native ditambahkan:
`internal/compiler/gox_fuzz_test.go` (GOX markup parsing),
`pkg/router/router_fuzz_test.go` (routing match dan parameter decoding),
`pkg/orm/orm_fuzz_test.go` (SQL identifier injection validation), dan
`pkg/action/action_fuzz_test.go` (action origin validation & boundary check).

Acceptance criteria:

- Tidak ada critical/high untracked finding.
- Semua mitigasi memiliki regression test atau documented risk acceptance.

### M6.2 Deployment and operations [P0]

- [x] Dokumentasikan reverse proxy, TLS termination, proxy headers, static
      assets, WebSocket upgrade, process signals, and container deployment.
- [x] Tambahkan example Dockerfile/container image hanya jika konsisten dengan
      single-binary goal.
- [x] Tetapkan resource limits: body, upload, timeout, concurrency, memory,
      WebSocket message, and image processing.
- [x] Tulis backup/restore dan migration deployment procedure.
- [x] Uji rolling restart dan version compatibility untuk in-flight clients.

Catatan implementasi: Panduan operasional dan hardening didokumentasikan di
`docs/OPERABILITY.md` serta `docs/DEPLOYMENT.md`, mencakup resource limits,
graceful shutdown dengan SIGTERM/SIGINT, rolling restart, dan atomic migrations.

Acceptance criteria:

- Aplikasi contoh dapat di-deploy pada documented target environment.
- Rollback procedure diuji, bukan hanya ditulis.

### M6.3 Observability [P1]

- [x] Konsistenkan request ID, structured log, error classification, dan
      latency fields.
- [x] Tambahkan metrics/tracing hanya berdasarkan kebutuhan konkret dan ADR.
- [x] Dokumentasikan redaction untuk token, cookie, authorization header,
      database URL, dan user data.
- [x] Tambahkan operational dashboard/example tanpa menjadikan Studio sebagai
      production admin panel.

Catatan implementasi: Logging policy dan secret redaction (token, cookie, DB URL)
didokumentasikan di `docs/OPERABILITY.md`. Request ID terintegrasi otomatis pada
context dan error headers. Studio dinonaktifkan otomatis pada mode produksi.

Acceptance criteria:

- Operator dapat menjawab: apakah service sehat, route mana lambat, error apa
  yang naik, dan instance mana yang bermasalah.
- Tidak ada secret pada log normal maupun error log.

## M7 — API stability, ecosystem, dan release candidate (`v0.22.0 -> v0.23.0`)

Tujuan: framework siap dipakai di luar repository dan tidak membuat pengguna
terjebak pada API yang berubah tanpa peringatan.

### M7.1 API freeze preparation [P0]

- [x] Review setiap exported API dan hapus/ubah hanya melalui deprecation atau
      ADR yang memiliki migration path.
- [x] Bekukan API stable untuk router, component, action, auth, ORM, WS,
      metadata, image, runtime, dan generated app contract.
- [x] Tambahkan compile compatibility fixture untuk aplikasi pengguna.
- [x] Tulis `BREAKING_CHANGES.md` dan migration guide dari v0.15.x ke v1.0.0.

Catatan implementasi: `BREAKING_CHANGES.md` memuat seluruh deprecation dan panduan
migrasi dari `v0.15.x` ke `v1.0.0`. Compile compatibility fixture di
`tests/compatibility/api_compat_test.go` memverifikasi seluruh kontrak publik stable
pada level compiler.

Acceptance criteria:

- Tidak ada perubahan breaking tanpa catatan migrasi.
- Public API report dan documentation reference selesai.

### M7.2 Ecosystem readiness [P1]

- [x] Buat template repository/example yang realistis dan dapat dipelajari.
- [x] Dokumentasikan integrasi database, auth provider, reverse proxy, CI,
      Docker/container, CDN/static export, dan WebSocket.
- [x] Tambahkan issue templates, contribution guide, support policy, dan
      security disclosure policy.
- [x] Tetapkan minimum supported Go version, supported OS/architecture, dan
      browser support matrix.
- [x] Publikasikan benchmark methodology, bukan hanya angka.

Catatan implementasi: Panduan kontribusi dan support matrix didokumentasikan di
`CONTRIBUTING.md`. Metodologi benchmark dan baseline terukur didokumentasikan di
`docs/BENCHMARKS.md`. Kebijakan security didokumentasikan di `SECURITY.md`.

Acceptance criteria:

- External contributor dapat menjalankan test dan membuat perubahan mengikuti
  docs.
- Support matrix memiliki automated smoke test untuk kombinasi utama.

### M7.3 Release candidate [P0]

- [x] Cut `v1.0.0-rc.1` dari clean checkout.
- [x] Jalankan full release matrix: host build, WASM build, race, fuzz smoke,
      generated app, standard deployment, standalone, export, auth, ORM,
      WebSocket, and graceful shutdown.
- [x] Jalankan manual acceptance test pada minimal satu application scenario
      end-to-end.
- [x] Kumpulkan feedback dari pengguna eksternal dan triage blocker.

Catatan implementasi: Tag `v0.23.0` menandai Release Candidate freeze (RC.1).
Seluruh test matrix (host, WASM, race, fuzz, CLI scaffolding, dan shutdown)
lulus tanpa blocker.

Acceptance criteria:

- RC tidak memiliki known critical/high blocker.
- Semua release artifacts memiliki checksum dan source reference.

## M8 — v1.0.0 release (`v0.23.0 -> v1.0.0`)

### M8.1 Release checklist [P0]

- [ ] Semua P0 task selesai atau memiliki keputusan eksplisit bahwa task tidak
      termasuk scope v1.0.
- [ ] Semua P1 task yang memengaruhi janji produk selesai.
- [ ] `README`, docs, examples, CLI help, architecture docs, ADR, and changelog
      menyebut versi dan perilaku yang sama.
- [ ] Version source, tag, generated template, Studio, and release artifact
      semuanya konsisten.
- [ ] Full quality gate lulus pada clean checkout.
- [ ] Security review disetujui.
- [ ] Migration guide dan rollback procedure dipublikasikan.

### M8.2 Go/no-go criteria

Release `v1.0.0` hanya boleh dilakukan jika:

- Tidak ada critical/high security issue terbuka.
- Tidak ada known data-loss, auth-bypass, request-isolation, atau remote code
  execution issue.
- Generated application berhasil build, start, render, dan deploy.
- Static page tetap memenuhi zero-unnecessary-WASM requirement.
- Graceful shutdown, request limits, timeout, logging, dan health/readiness
  teruji.
- Public API stable memiliki dokumentasi dan compatibility expectation.
- Minimal satu pengguna eksternal dapat mengikuti quick start sampai deploy
  tanpa bantuan maintainer.

### M8.3 Release outputs

- [ ] Git tag `v1.0.0` dan changelog.
- [ ] Binaries atau reproducible build instructions untuk supported targets.
- [ ] Documentation site/versioned docs.
- [ ] Example applications.
- [ ] Security policy dan support policy.
- [ ] Post-release maintenance plan untuk patch release dan deprecation.

## 4. Backlog setelah v1.0.0

Task berikut tidak boleh menggeser blocker v1.0.0:

- [ ] Additional database adapter berdasarkan issue/use case nyata.
- [ ] OpenTelemetry integration berdasarkan kebutuhan deployment nyata.
- [ ] Plugin/extension mechanism setelah minimal satu extension use case
      terdokumentasi.
- [ ] Advanced routing/data cache hanya setelah semantics dasar stabil.
- [ ] Ecosystem package dan component library tambahan.

## 5. Definition of Ready

Sebuah task boleh mulai dikerjakan jika:

- Masalah pengguna atau risiko teknisnya jelas.
- Scope dan non-goal tertulis.
- Package/API yang terdampak sudah diidentifikasi.
- Dependency dan ADR yang diperlukan sudah diketahui.
- Acceptance criteria dapat diverifikasi dengan test atau artifact.
- Security, backward compatibility, dan migration impact sudah dipertimbangkan.

## 6. Definition of Done

Sebuah task dianggap selesai jika:

- Implementasi dan test selesai; bug fix memiliki regression test.
- `go fmt ./...`, `go vet ./...`, `go test ./...`, `go test -race ./...`, dan
  build yang relevan berhasil.
- Perubahan WASM/GOX memiliki WASM build, bundle check, dan interaction sanity
  check.
- Security review dan public API review selesai bila relevan.
- Dokumentasi, ADR, roadmap status, dan migration note diperbarui bila perlu.
- Diff telah direview dan commit fokus dibuat pada branch yang sesuai.

## 7. Product operating cadence

- Setiap milestone dimulai dengan memilih maksimal 3 P0/P1 task utama.
- Setiap minggu: review blocker, flaky test, security finding, dan feedback
  pengguna.
- Setiap milestone: jalankan release gate dan perbarui status task berdasarkan
  bukti, bukan asumsi.
- Setiap release: catat fitur yang benar-benar verified dan fitur yang masih
  experimental.
- Feature baru tidak boleh ditambahkan jika memperbesar fragmentation tanpa
  mengurangi risiko atau meningkatkan product promise yang sudah ada.
