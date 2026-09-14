# GOX Syntax and Compiler Specification

This document defines the formal grammar, semantics, transformation rules, and compiler diagnostic behavior of GOX (`.gox`) templates in the GoKS framework.

## 1. Overview

GOX brings JSX-like expressive templating into pure Go. GOX files are standard Go source files where `return (...)` blocks contain HTML/XML-style component markup that is transpiled ahead-of-time (or during `goks dev` / `goks build`) into Go method calls on `pkg/html` and `pkg/component`.

GOX is:
- **Zero Runtime Dependencies**: No Node.js or JavaScript runtime required; transpilation is implemented in pure Go.
- **Strictly Typed**: All expressions within `{...}` are standard Go expressions evaluated by the Go compiler.
- **Deterministic**: Attribute and property ordering is sorted deterministically at compile time.
- **Source-Mapped Diagnostics**: Compiler errors report exact `.gox` file paths, lines, columns, and visual source snippets.

---

## 2. Grammar & Transformations

### 2.1 HTML Elements

Standard HTML elements map to `html.<Tag>(children...).Attr(...)`:

```html
<div class="container" id="main">
    <h1>Hello GoKS</h1>
</div>
```

Transpiles to:
```go
html.Div(html.H1("Hello GoKS")).Class("container").ID("main")
```

### 2.2 Fragments

Fragments group multiple children without adding an extra DOM wrapper:

Shorthand syntax:
```html
<>
    <header>Header</header>
    <main>Content</main>
</>
```

Or explicit syntax:
```html
<Fragment>
    <header>Header</header>
    <main>Content</main>
</Fragment>
```

Transpiles to:
```go
component.Fragment(html.Header("Header"), html.Main("Content"))
```

### 2.3 Component Instantiation

Custom components are referenced with dot notation `<package.Component ... />`:

```html
<c.Hero title="GoKS Framework" count={42} active={true} />
```

Transpiles to:
```go
component.C(&c.Hero{active: true, count: 42, title: "GoKS Framework"})
```

### 2.4 Event Handlers

Attributes prefixed with `on` or `On` (e.g. `onClick`, `onInput`, `onSubmit`) bind event handlers:

```html
<button class="btn" onClick={handleClick}>
    Click Me
</button>
```

Transpiles to:
```go
html.Button("Click Me").Class("btn").On("onClick", handleClick)
```

### 2.5 Dynamic Expressions

Dynamic Go expressions are wrapped in `{...}`:
- As child nodes: wrapped in `component.Any(expr)`
- As attribute values: passed directly to the corresponding attribute or property setter

```html
<div class={themeClass}>
    {user.Name}
    <span>{formatDate(user.CreatedAt)}</span>
</div>
```

Transpiles to:
```go
html.Div(component.Any(user.Name), html.Span(component.Any(formatDate(user.CreatedAt)))).Class(themeClass)
```

### 2.6 HTML Comments

HTML comments `<!-- ... -->` are safely ignored by the compiler and omitted from the generated Go VDOM tree.

---

## 3. Error Diagnostics

When syntax errors occur, the compiler generates a `*compiler.CompileError` pointing directly to the `.gox` source:

```
app/pages/home.gox:14:5: compile error: unclosed 'return (' block: matching ')' not found
  14 |     return (
     |     ^
```

```
app/pages/about.gox:8:9: compile error: failed to parse GOX markup: XML syntax error on line 3: element <p> closed by </div>
   8 |         <p>Invalid nesting</div>
     |         ^
```
