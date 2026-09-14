# GoKS Public API Inventory & Compatibility Policy

Baseline Version: v0.16.0  
Status: Active  

Dokumen ini adalah inventaris resmi seluruh exported API pada paket `pkg/*` dan `runtime/*` framework GoKS. Setiap perubahan pada exported API harus mematuhi kebijakan kompatibilitas di bawah ini.

---

## 1. Stability Classification (Klasifikasi Stabilitas)

Setiap exported identifier di GoKS dikelompokkan ke dalam salah satu dari tiga kategori:

| Kategori | Kontrak Kompatibilitas | Contoh Paket |
| :--- | :--- | :--- |
| **Stable** | Terikat garansi backward compatibility SemVer. Tidak boleh diubah signature/perilakunya secara breaking tanpa major release (`vX.0.0`), deprecation cycle minimal 1 minor release, dan ADR resmi. | `pkg/router`, `pkg/action`, `pkg/component`, `pkg/orm`, `pkg/auth`, `pkg/rbac`, `pkg/metadata` |
| **Experimental** | Fitur baru atau evolutif. Dapat menerima penyempurnaan signature pada minor release jika disertai release notes dan migration guide yang jelas. | `pkg/ws`, `pkg/store`, `pkg/image`, `pkg/rpc`, `runtime/server` |
| **Internal / Tooling Contract** | Digunakan oleh CLI scaffolding dan generated code, tidak ditujukan untuk dipanggil langsung oleh aplikasi pengguna akhir. | `runtime/client`, `pkg/studio`, helper internal |

---

## 2. Breaking Change & Deprecation Policy

1. **Prinsip Garansi Publik**:
   - Fungsi atau tipe yang berstatus `Stable` tidak boleh dihapus atau diubah signature-nya secara breaking dalam versi minor yang sama.
   - Jika suatu API hendak digantikan, API lama harus ditandai dengan komentar `// Deprecated: gunakan NewFunc sebagai gantinya.` dan tetap dipertahankan minimal hingga rilis major berikutnya.
2. **Prosedur Perubahan Breaking**:
   - Tulis Architecture Decision Record (ADR) di `docs/adr/`.
   - Dokumentasikan perubahan dan panduan migrasi di `BREAKING_CHANGES.md`.
   - Sediakan compile-level test dan compiler diagnostic jika relevan.
3. **Format Catatan Migrasi (Migration Note)**:
   ```markdown
   ### [Deprecated / Changed API Name]
   - **Status**: Deprecated di v0.x / Breaking di v1.x
   - **Alasan**: <Penjelasan rasional arsitektural atau keamanan>
   - **Sebelum**:
     ```go
     ctx.OldCall()
     ```
   - **Sesudah**:
     ```go
     ctx.NewCall()
     ```
   ```

---

## 3. Public API Inventory

### 3.1 `pkg/router` (Stable)
- `type Router`: Core HTTP router GoKS berbasis Radix Trie.
- `func New() *Router`: Menginisialisasi router baru.
- `func (r *Router) GET / POST / PUT / DELETE / PATCH / OPTIONS / HEAD(path string, handler Handler)`: Mendaftarkan route HTTP.
- `func (r *Router) Group(prefix string, middlewares ...MiddlewareFunc) *Group`: Membuat route group.
- `func (r *Router) Use(middlewares ...MiddlewareFunc)`: Menambahkan middleware global.
- `func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request)`: Mengimplementasikan `http.Handler`.
- `type Context`: Request context yang membungkus `*http.Request` dan `http.ResponseWriter`.
- `func (c *Context) Param(key string) string`: Mengambil path parameter.
- `func (c *Context) Query(key string) string`: Mengambil query parameter.
- `func (c *Context) Status(code int) *Context`: Menetapkan HTTP status code.
- `func (c *Context) JSON(data any) error`: Mengirimkan response JSON.
- `func (c *Context) HTML(html string) error`: Mengirimkan response HTML.
- `func (c *Context) Text(text string) error`: Mengirimkan response plain text.
- `func (c *Context) Redirect(url string, code ...int) error`: Mengirimkan redirect response.
- `func (c *Context) Bind(v any) error`: Mem-bind JSON body ke struct target.
- `type Handler`: `func(ctx *Context) error`.
- `type MiddlewareFunc`: `func(Handler) Handler`.

### 3.2 `pkg/action` (Stable)
- `func Register(name string, fn Action)`: Mendaftarkan server action progresif.
- `func Handler() http.HandlerFunc`: HTTP endpoint handler untuk mengeksekusi server actions via POST JSON maupun form submission fallback.
- `func URL(name string) string`: Menghasilkan URL endpoint pemanggilan action.
- `func RegisteredActions() []string`: Mengembalikan daftar nama server action yang terdaftar.
- `type Action`: `func(ctx *Context) (any, error)`.
- `type Context`: Context pembungkus payload input action dan metadata request.
- `type Response`: Payload balasan terstandarisasi server action.

### 3.3 `pkg/component` (Stable)
- `type Node`: Virtual DOM representation (`ElementNode`, `TextNode`, `ComponentNode`).
- `func H(tag string, props Props, children ...*Node) *Node`: Membuat virtual DOM element node.
- `func Text(content string) *Node`: Membuat text node.
- `func Fragment(children ...*Node) *Node`: Membuat fragment node.
- `func Suspense(props SuspenseProps) *Node`: Streaming SSR suspense boundary.
- `func UseState[T any](initial T) (T, func(T))`: Hook reactive state per-component.
- `func RenderToString(node *Node) string`: Merender virtual DOM menjadi HTML string untuk SSR.
- `func NeedsHydration(node *Node) bool`: Memeriksa apakah node memerlukan WASM hydration.

### 3.4 `pkg/orm` (Stable)
- `type Database`: Wrapper thread-safe database connection berbasis `database/sql`.
- `func Connect(driver, dsn string) (*Database, error)`: Membuka koneksi database terkelola.
- `func OpenSQLite(path ...string) (*Database, error)`: Membuka koneksi SQLite terkelola dengan pure-Go driver bawaan.
- `type Builder[T any]`: Query builder dengan method chaining aman:
  - `Query[T any](db ...*Database) *Builder[T]`
  - `Where(condition string, args ...any) *Builder[T]`
  - `Limit(n int) *Builder[T]`
  - `Offset(n int) *Builder[T]`
  - `OrderBy(col string) *Builder[T]`
  - `Find() ([]T, error)`
  - `First() (*T, error)`
  - `Count() (int64, error)`
- `func Create[T any](db *Database, m *T) error`
- `func Save[T any](db *Database, m *T) error`
- `func Delete[T any](db *Database, m *T) error`
- `func HardDelete[T any](db *Database, m *T) error`
- `func Migrate(db *Database, migrations []Migration) error`
- `func Rollback(db *Database, migrations []Migration, steps int) error`

### 3.5 `pkg/auth` & `pkg/auth/oauth` (Stable)
- `type Manager`: Session auth manager.
- `func New(ttl time.Duration) *Manager`: Inisialisasi session auth manager.
- `func (m *Manager) Login(w http.ResponseWriter, r *http.Request, user *User) (*Session, error)`
- `func (m *Manager) Logout(w http.ResponseWriter, r *http.Request)`
- `func (m *Manager) SessionFromRequest(r *http.Request) (*Session, bool)`
- `type JWT`: JWT auth manager.
- `func NewJWT(secret string, ttl time.Duration) *JWT`
- `func JWTMiddleware() router.MiddlewareFunc`
- `func Required() router.MiddlewareFunc`
- `func RequireRole(roles ...string) router.MiddlewareFunc`
- `pkg/auth/oauth`: Provider OAuth2 dengan PKCE (GitHub, Google) dengan timing-safe state comparison.

### 3.6 `pkg/rbac` (Stable)
- `type Registry`: Role-Based Access Control registry.
- `func NewRegistry() *Registry`: Inisialisasi registry baru.
- `func (reg *Registry) Define(roleName string, perms ...Permission)`
- `func (reg *Registry) Has(roleName string, perm Permission) bool`
- `func (reg *Registry) UserHas(user *auth.User, perm Permission) bool`
- `func Can(perm Permission) router.MiddlewareFunc`: Middleware proteksi endpoint berdasarkan permission.
- `func HasRole(roles ...string) router.MiddlewareFunc`: Middleware proteksi endpoint berdasarkan role.

### 3.7 `pkg/metadata` (Stable)
- `type Metadata`: Struktur SEO metadata (Title, Description, OpenGraph, Twitter card, Canonical, Icons).
- `func RenderHTML(m Metadata) string`: Merender tag HTML `<meta>` dan `<title>`.
- `func ExtractFromTree(node *component.Node) Metadata`: Mengekstrak metadata dari tree component.
- `func Merge(parent, child Metadata) Metadata`: Menggabungkan metadata parent dan child page.

### 3.8 `pkg/ws` (Experimental)
- `type Hub`: Thread-safe WebSocket client manager.
- `func NewHub() *Hub`
- `func (h *Hub) Broadcast(msg []byte)`
- `func (h *Hub) Handler() http.HandlerFunc`
- `type EventHub`: Typed event router di atas Hub.
- `func NewEventHub() *EventHub`
- `func (h *EventHub) On(event string, handler EventHandler)`
- `func (h *EventHub) BroadcastEvent(event string, data any)`
- `func (h *EventHub) Join(room string, client *Client)`
- `func (h *EventHub) Leave(room string, client *Client)`

### 3.9 `pkg/store` (Experimental)
- `type Store[T any]`: Reactive state container untuk WASM client.
- `func New[T any](initial T) *Store[T]`
- `func (s *Store[T]) Get() T`
- `func (s *Store[T]) Set(val T)`
- `func (s *Store[T]) Subscribe(fn func(T)) (unsubscribe func())`

### 3.10 `pkg/cache`, `pkg/env`, `pkg/font`, `pkg/html`, `pkg/image`, `pkg/rpc` (Stable / Experimental)
- `pkg/cache`: In-memory cache dengan TTL expiry (`New(ttl)`).
- `pkg/env`: Pembaca variabel lingkungan dan `.env` loader (`Load()`, `Get()`).
- `pkg/font/google`: Loader web font Google Fonts yang dioptimalkan untuk SSR.
- `pkg/html`: String escaping dan sanitasi tag HTML.
- `pkg/image`: Komponen gambar responsif dan image optimizer.
- `pkg/rpc`: Wire protocol client-server WASM bridge.

### 3.11 `runtime/server` (Experimental / Glue)
- `type Server`: Instance HTTP server GoKS production & dev.
- `func NewServer(cfg Config) *Server`
- `func (s *Server) Start() error`
- `func (s *Server) Shutdown(ctx context.Context) error`

### 3.12 `runtime/client` (Internal / WASM Only)
- Client-side entrypoint untuk bootstrap WebAssembly, hydration virtual DOM ke DOM browser, dan event delegation.
