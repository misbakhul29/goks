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

- [ ] Tetapkan dan dokumentasikan urutan middleware: Logger, CORS, Secure,
      RequestID, Compress, MaxBytes, Timeout, auth, dan application middleware.
- [ ] Uji bahwa semua route menerima policy yang sama, termasuk API route dan
      error response.
- [ ] Tetapkan default yang aman untuk CORS, timeout, body limit, dan security
      headers.
- [ ] Dokumentasikan cara override default tanpa menghapus kontrol keamanan.

Acceptance criteria:

- Ada integration test untuk urutan middleware dan short-circuit behavior.
- Error response tidak membocorkan stack trace atau secret pada production.

### M1.2 Request lifecycle dan graceful shutdown [P0]

- [ ] Pastikan context request diteruskan sampai handler, action, ORM, dan
      external call.
- [ ] Implementasikan atau verifikasi graceful shutdown dengan deadline.
- [ ] Drain in-flight HTTP request dan tutup WebSocket connection dengan bersih.
- [ ] Uji SIGTERM/SIGINT pada standard dan standalone build.

Acceptance criteria:

- Shutdown test membuktikan request aktif tidak dipotong sebelum deadline.
- Tidak ada goroutine leak yang terdeteksi pada test lifecycle.

### M1.3 Structured logging dan health [P1]

- [ ] Tetapkan structured log schema: timestamp, level, request ID, method,
      path, status, latency, dan error class.
- [ ] Pisahkan log developer dari log production.
- [ ] Sediakan pola endpoint health/readiness yang tidak bergantung pada Studio.
- [ ] Dokumentasikan liveness, readiness, dependency failure, dan shutdown.

Acceptance criteria:

- Log dapat diproses mesin tanpa parsing string ad-hoc.
- Health endpoint tidak membocorkan konfigurasi atau credential.

### M1.4 Concurrency/load baseline [P1]

- [ ] Tambahkan concurrent request test untuk `runtime/server` dan router.
- [ ] Audit `pkg/ws` untuk goroutine leak, slow consumer, concurrent write,
      room lifecycle, dan authorization.
- [ ] Audit `pkg/store` untuk race, subscription leak, dan update ordering.
- [ ] Tambahkan benchmark baseline untuk router, component render, dan WS hub.

Acceptance criteria:

- Race test bersih pada workload concurrent yang terdokumentasi.
- Slow client tidak dapat memblokir seluruh hub.
- Baseline benchmark disimpan agar regression terukur.

## M2 — Routing, API routes, dan error model (`v0.17.0 -> v0.18.0`)

Tujuan: routing file-system dan programmatic menjadi predictable dan mudah
debug.

### M2.1 Route semantics [P0]

- [ ] Tetapkan precedence static, dynamic, catch-all, API, layout, dan not-found.
- [ ] Uji nested layout dan inheritance metadata.
- [ ] Uji route parameter decoding, invalid parameter, encoded slash, query
      parameter, trailing slash, method mismatch, dan duplicate route.
- [ ] Uji route groups atau dokumentasikan bahwa fitur tersebut belum menjadi
      bagian API v1.

Acceptance criteria:

- Semua precedence rule tertulis dan memiliki table-driven test.
- Conflict route menghasilkan compile/generate error yang jelas.

### M2.2 API route contract [P0]

- [ ] Tetapkan handler signature, supported methods, status code, headers,
      JSON binding, content negotiation, dan error response.
- [ ] Pastikan API route tidak masuk bundle WASM.
- [ ] Uji body limit, malformed JSON, timeout, panic recovery, dan context cancel.
- [ ] Dokumentasikan CORS/auth policy untuk API route.

Acceptance criteria:

- API example dari `goks new` dapat di-build dan diuji tanpa edit manual.
- Error API konsisten dan tidak membocorkan detail internal.

### M2.3 Error and observability model [P1]

- [ ] Definisikan not-found, method-not-allowed, validation, auth, forbidden,
      conflict, dan internal error mapping.
- [ ] Tambahkan request ID ke response dan log untuk error.
- [ ] Buat dev overlay yang menampilkan informasi berguna tanpa muncul di
      production response.

Acceptance criteria:

- User dapat membedakan error routing, compile, application, dan dependency.
- Error contract didokumentasikan untuk page dan API.

## M3 — ORM, data access, auth, dan security (`v0.18.0 -> v0.19.0`)

Tujuan: data dan identity layer dapat dipercaya untuk aplikasi produksi.

### M3.1 ORM safety and semantics [P0]

- [ ] Audit seluruh query builder: parameterization, identifier validation,
      ordering, pagination, joins, aggregates, dan raw query escape hatch.
- [ ] Dokumentasikan transaction semantics, commit/rollback, nested behavior,
      connection ownership, dan context cancellation.
- [ ] Uji pool limits, concurrent transaction, failed transaction, dan retry
      policy yang memang didukung.
- [ ] Dokumentasikan soft-delete scope, restore, timestamps, hooks, dan update
      behavior.

Acceptance criteria:

- Tidak ada input user yang dirangkai menjadi SQL tanpa validasi/parameterisasi.
- Integration test menggunakan database nyata untuk perilaku ORM penting.
- Race test dan failure-path test lulus.

### M3.2 Migrations [P0]

- [ ] Tetapkan naming/versioning migration dan checksum policy.
- [ ] Uji apply, rollback, partial failure, transaction atomicity, lock,
      concurrent deploy, dan status drift.
- [ ] Pastikan `goks db` aman terhadap path traversal dan command injection.
- [ ] Sediakan migration guide untuk production deploy.

Acceptance criteria:

- Dua proses migration bersamaan tidak merusak schema.
- Failure dapat dilanjutkan atau dipulihkan melalui prosedur terdokumentasi.

### M3.3 Database adapter truthfulness [P1]

- [ ] Verifikasi dukungan SQLite, PostgreSQL, dan MySQL berdasarkan source,
      driver, test, dan generated example.
- [ ] Jika adapter belum benar-benar didukung, koreksi README sebelum v1.0.
- [ ] Tambahkan adapter hanya jika ada use case konkret, ADR, dan integration
      test matrix.

Catatan implementasi: `pkg/orm` mendaftarkan driver pure-Go
`modernc.org/sqlite` melalui `pkg/orm/sqlite_driver.go` sehingga `orm.OpenSQLite`
dan `goks db status` bekerja tanpa konfigurasi. File tersebut memakai build
constraint `!js && !wasm`: driver sengaja dikecualikan dari build WASM karena
`modernc.org/libc` tidak mendukung js/wasm dan SQLite hanya relevan untuk server.
PostgreSQL dan MySQL masih bergantung pada driver aplikasi dan belum diverifikasi.

Acceptance criteria:

- README hanya menyebut adapter yang build dan test-nya terbukti.
- Connection string, credential, dan error database tidak masuk log/artifact.

### M3.4 Authentication and authorization [P0]

- [ ] Dokumentasikan session/JWT threat model dan kapan masing-masing dipakai.
- [ ] Uji cookie flags, expiry, rotation, revocation, replay, CSRF, fixation,
      token leakage, dan logout.
- [ ] Uji RBAC deny-by-default, role inheritance jika ada, dan authorization
      pada page, API, action, WebSocket, dan Studio.
- [ ] Uji OAuth state, PKCE, redirect allowlist, provider error, token expiry,
      account linking, dan callback failure.
- [ ] Pastikan secret hanya berasal dari environment/secret manager dan tidak
      pernah masuk generated source atau diagnostics.

Acceptance criteria:

- Protected resource memiliki test allow dan deny.
- Security-sensitive default aktif dan cara menonaktifkannya terdokumentasi.

## M4 — GOX, component model, SSR, dan WASM (`v0.19.0 -> v0.20.0`)

Tujuan: frontend GoKS stabil, terukur, dan tidak mengejutkan developer.

### M4.1 GOX compiler correctness [P0]

- [ ] Buat grammar/syntax reference yang sama dengan parser aktual.
- [ ] Tambahkan golden test untuk element, attribute, expression, component,
      event handler, children, fragments, comments, dan malformed input.
- [ ] Map compile error ke file `.gox`, line, column, dan source snippet.
- [ ] Pastikan generated Go deterministik dan tidak bergantung pada map order.
- [ ] Uji import, package alias, reserved word, path traversal, dan generated
      identifier collision.

Acceptance criteria:

- Error utama menunjuk lokasi `.gox`, bukan hanya generated `.go`.
- Rebuild source yang sama menghasilkan output yang sama.

### M4.2 Component and hooks contract [P0]

- [ ] Tetapkan lifecycle render, mount, update, unmount, dan error behavior.
- [ ] Enforce hook call-order invariant dan beri error yang jelas.
- [ ] Uji nested component, keyed children, event handler, state update,
      concurrent render, dan cleanup.
- [ ] Dokumentasikan batasan server render versus client hydration.
- [ ] Audit global state/hook state terhadap request isolation dan data leakage.

Acceptance criteria:

- Test component bersih dengan race detector.
- State satu request/user tidak pernah terlihat di request/user lain.

### M4.3 SSR, streaming, Suspense [P1]

- [ ] Uji shell, fallback, resolved content, error boundary, client disconnect,
      timeout, ordering, cancellation, dan backpressure.
- [ ] Pastikan streaming tidak mengirim data sensitif sebelum authorization.
- [ ] Dokumentasikan kapan streaming dipakai dan kapan response biasa lebih tepat.

Acceptance criteria:

- Streaming test dapat memverifikasi chunk order dan cancellation.
- No-JS browser tetap menerima konten yang usable.

### M4.4 Islands and hydration [P0]

- [ ] Buat fixture static page yang menghasilkan 0 KB WASM.
- [ ] Buat fixture interactive page yang hanya memuat island yang diperlukan.
- [ ] Uji SSR-to-hydration identity, event handler, state initialization,
      navigation, failure recovery, dan duplicate hydration.
- [ ] Tambahkan bundle-size budget dan regression report ke CI.
- [ ] Pastikan API route dan server-only code tidak masuk client bundle.

Acceptance criteria:

- Static fixture memiliki automated zero-WASM assertion.
- Hydration fixture memiliki browser-level atau equivalent DOM interaction test.

### M4.5 TinyGo and store [P1]

- [ ] Verifikasi parity antara standard Go WASM dan TinyGo.
- [ ] Dokumentasikan unsupported API dan fallback jika parity belum tercapai.
- [ ] Ukur bundle size, startup time, memory, dan build time.
- [ ] Dokumentasikan `pkg/store`: subscription, rerender trigger, batching,
      unsubscribe, dan ordering.

Acceptance criteria:

- TinyGo hanya dipromosikan sebagai supported jika feature matrix lulus.
- Store memiliki test untuk lifecycle dan concurrent update.

## M5 — CLI, generated app, dan developer experience (`v0.20.0 -> v0.21.0`)

Tujuan: programmer dapat belajar dan mengirim aplikasi tanpa membaca source
framework terlebih dahulu.

### M5.1 `goks new` golden project [P0]

- [ ] Generated project memiliki `go.mod`, page, layout, component, API,
      middleware, model, dan README yang benar-benar build.
- [ ] Tambahkan end-to-end test: `goks new -> go mod tidy -> build -> test ->
      start -> request page`.
- [ ] Pastikan generated version, Go version, module path, dan import path benar.
- [ ] Validasi nama app, path, module, overwrite protection, dan permissions.

Acceptance criteria:

- Fresh generated project build tanpa edit manual.
- Template tidak mengandung placeholder yang menghasilkan code rusak.

### M5.2 `goks dev` reliability [P0]

- [ ] Uji watch untuk `.gox`, `.go`, CSS/Tailwind, config, route, dan deleted file.
- [ ] Uji debounce, duplicate event, failed compile, recovery, and restart.
- [ ] Pisahkan overlay GOX, Go build, Tailwind, dan runtime error.
- [ ] Pastikan dev server tidak meninggalkan process atau temporary workspace.
- [ ] Ukur cold start dan hot reload latency dengan baseline.

Acceptance criteria:

- Save setelah compile error memulihkan server tanpa restart manual.
- Overlay tidak aktif pada production build.

### M5.3 Build, standalone, start, dan export [P0]

- [ ] Uji standard build, standalone build, `start`, dan SSG/export dari clean
      checkout.
- [ ] Verifikasi asset embedding: HTML, CSS, WASM, images, fonts, dan public.
- [ ] Uji binary pada working directory berbeda tanpa source dependency.
- [ ] Uji reproducibility, missing asset, invalid flag, output collision, dan
      cross-platform path handling.
- [ ] Dokumentasikan runtime file dependency yang memang masih diperlukan.

Acceptance criteria:

- `goks build --standalone` dapat dijalankan pada directory baru.
- SSG output dapat diserve sebagai static files dan memiliki link/asset yang
  benar.

### M5.4 CLI and LSP contract [P1]

- [ ] Dokumentasikan seluruh command, subcommand, flag, default, exit code,
      dan error message.
- [ ] Uji `generate`, `page`, `ui`, `db`, `studio`, `export`, dan `lsp`.
- [ ] Tambahkan completion/hover/diagnostic/formatting test untuk syntax GOX.
- [ ] Pastikan CLI error dapat dipakai di script: non-zero exit, no ANSI-only
      information, dan output yang konsisten.

Acceptance criteria:

- `--help` dan docs tidak berbeda dari implementasi.
- LSP tidak crash pada malformed atau incomplete `.gox`.

### M5.5 Documentation and examples [P0]

- [ ] Tulis getting started sampai deployment dalam urutan yang dapat diikuti.
- [ ] Buat examples minimal: blog/CRUD, auth/RBAC, API, action, WebSocket,
      upload/image, static export, dan standalone deployment.
- [ ] Setiap example harus masuk CI atau memiliki verification script.
- [ ] Tambahkan troubleshooting untuk Go/WASM/TinyGo/Tailwind/database.
- [ ] Tulis comparison yang faktual; hapus klaim yang belum terukur.

Acceptance criteria:

- New developer dapat menjalankan hello world dan production build dari clean
  checkout mengikuti docs saja.
- Link, command, version, dan output contoh diverifikasi.

## M6 — Production hardening dan operability (`v0.21.0 -> v0.22.0`)

Tujuan: aplikasi GoKS dapat di-deploy, diamati, dan dipulihkan secara aman.

### M6.1 Security release gate [P0]

- [ ] Lakukan threat model review terhadap HTTP, template/HTML, WASM bridge,
      action/RPC, auth, ORM, file serving, image fetch, WebSocket, CLI, dan
      Studio.
- [ ] Tambahkan fuzz test untuk parser GOX, router, query builder, JSON/action
      input, dan security-sensitive decoders.
- [ ] Audit SSRF, path traversal, XSS, CSRF, request smuggling, zip/tar bomb,
      secret exposure, and denial-of-service limits.
- [ ] Review dependency provenance, checksums, and release artifact integrity.

Acceptance criteria:

- Tidak ada critical/high untracked finding.
- Semua mitigasi memiliki regression test atau documented risk acceptance.

### M6.2 Deployment and operations [P0]

- [ ] Dokumentasikan reverse proxy, TLS termination, proxy headers, static
      assets, WebSocket upgrade, process signals, and container deployment.
- [ ] Tambahkan example Dockerfile/container image hanya jika konsisten dengan
      single-binary goal.
- [ ] Tetapkan resource limits: body, upload, timeout, concurrency, memory,
      WebSocket message, and image processing.
- [ ] Tulis backup/restore dan migration deployment procedure.
- [ ] Uji rolling restart dan version compatibility untuk in-flight clients.

Acceptance criteria:

- Aplikasi contoh dapat di-deploy pada documented target environment.
- Rollback procedure diuji, bukan hanya ditulis.

### M6.3 Observability [P1]

- [ ] Konsistenkan request ID, structured log, error classification, dan
      latency fields.
- [ ] Tambahkan metrics/tracing hanya berdasarkan kebutuhan konkret dan ADR.
- [ ] Dokumentasikan redaction untuk token, cookie, authorization header,
      database URL, dan user data.
- [ ] Tambahkan operational dashboard/example tanpa menjadikan Studio sebagai
      production admin panel.

Acceptance criteria:

- Operator dapat menjawab: apakah service sehat, route mana lambat, error apa
  yang naik, dan instance mana yang bermasalah.
- Tidak ada secret pada log normal maupun error log.

## M7 — API stability, ecosystem, dan release candidate (`v0.22.0 -> v0.23.0`)

Tujuan: framework siap dipakai di luar repository dan tidak membuat pengguna
terjebak pada API yang berubah tanpa peringatan.

### M7.1 API freeze preparation [P0]

- [ ] Review setiap exported API dan hapus/ubah hanya melalui deprecation atau
      ADR yang memiliki migration path.
- [ ] Bekukan API stable untuk router, component, action, auth, ORM, WS,
      metadata, image, runtime, dan generated app contract.
- [ ] Tambahkan compile compatibility fixture untuk aplikasi pengguna.
- [ ] Tulis `BREAKING_CHANGES.md` dan migration guide dari v0.15.x ke v1.0.0.

Acceptance criteria:

- Tidak ada perubahan breaking tanpa catatan migrasi.
- Public API report dan documentation reference selesai.

### M7.2 Ecosystem readiness [P1]

- [ ] Buat template repository/example yang realistis dan dapat dipelajari.
- [ ] Dokumentasikan integrasi database, auth provider, reverse proxy, CI,
      Docker/container, CDN/static export, dan WebSocket.
- [ ] Tambahkan issue templates, contribution guide, support policy, dan
      security disclosure policy.
- [ ] Tetapkan minimum supported Go version, supported OS/architecture, dan
      browser support matrix.
- [ ] Publikasikan benchmark methodology, bukan hanya angka.

Acceptance criteria:

- External contributor dapat menjalankan test dan membuat perubahan mengikuti
  docs.
- Support matrix memiliki automated smoke test untuk kombinasi utama.

### M7.3 Release candidate [P0]

- [ ] Cut `v1.0.0-rc.1` dari clean checkout.
- [ ] Jalankan full release matrix: host build, WASM build, race, fuzz smoke,
      generated app, standard deployment, standalone, export, auth, ORM,
      WebSocket, and graceful shutdown.
- [ ] Jalankan manual acceptance test pada minimal satu application scenario
      end-to-end.
- [ ] Kumpulkan feedback dari pengguna eksternal dan triage blocker.

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
