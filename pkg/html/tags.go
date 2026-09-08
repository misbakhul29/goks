package html

import "github.com/misbakhulmunir/goks/pkg/component"

// Div creates a <div> element.
func Div(children ...any) *component.Node {
	return Element("div", children...)
}

// Main creates a <main> element.
func Main(children ...any) *component.Node {
	return Element("main", children...)
}

// Section creates a <section> element.
func Section(children ...any) *component.Node {
	return Element("section", children...)
}

// Header creates a <header> element.
func Header(children ...any) *component.Node {
	return Element("header", children...)
}

// Footer creates a <footer> element.
func Footer(children ...any) *component.Node {
	return Element("footer", children...)
}

// Nav creates a <nav> element.
func Nav(children ...any) *component.Node {
	return Element("nav", children...)
}

// H1 creates an <h1> element.
func H1(children ...any) *component.Node {
	return Element("h1", children...)
}

// H2 creates an <h2> element.
func H2(children ...any) *component.Node {
	return Element("h2", children...)
}

// H3 creates an <h3> element.
func H3(children ...any) *component.Node {
	return Element("h3", children...)
}

// P creates a <p> element.
func P(children ...any) *component.Node {
	return Element("p", children...)
}

// Span creates a <span> element.
func Span(children ...any) *component.Node {
	return Element("span", children...)
}

// Strong creates a <strong> element.
func Strong(children ...any) *component.Node {
	return Element("strong", children...)
}

// A creates an <a> element.
func A(children ...any) *component.Node {
	return Element("a", children...)
}

// Img creates an <img> element.
func Img(children ...any) *component.Node {
	return Element("img", children...)
}

// Form creates a <form> element.
func Form(children ...any) *component.Node {
	return Element("form", children...)
}

// Input creates an <input> element.
func Input(children ...any) *component.Node {
	return Element("input", children...)
}

// Button creates a <button> element.
func Button(children ...any) *component.Node {
	return Element("button", children...)
}

// Label creates a <label> element.
func Label(children ...any) *component.Node {
	return Element("label", children...)
}

// Ul creates a <ul> element.
func Ul(children ...any) *component.Node {
	return Element("ul", children...)
}

// Li creates an <li> element.
func Li(children ...any) *component.Node { return Element("li", children...) }

// H4 creates an <h4> element.
func H4(children ...any) *component.Node { return Element("h4", children...) }
func H5(children ...any) *component.Node { return Element("h5", children...) }
func H6(children ...any) *component.Node { return Element("h6", children...) }

func Article(children ...any) *component.Node { return Element("article", children...) }
func Aside(children ...any) *component.Node { return Element("aside", children...) }
func Figure(children ...any) *component.Node { return Element("figure", children...) }
func Figcaption(children ...any) *component.Node { return Element("figcaption", children...) }

func Ol(children ...any) *component.Node { return Element("ol", children...) }
func Table(children ...any) *component.Node { return Element("table", children...) }
func Thead(children ...any) *component.Node { return Element("thead", children...) }
func Tbody(children ...any) *component.Node { return Element("tbody", children...) }
func Tr(children ...any) *component.Node { return Element("tr", children...) }
func Th(children ...any) *component.Node { return Element("th", children...) }
func Td(children ...any) *component.Node { return Element("td", children...) }

func I(children ...any) *component.Node { return Element("i", children...) }
func B(children ...any) *component.Node { return Element("b", children...) }
func U(children ...any) *component.Node { return Element("u", children...) }
func Em(children ...any) *component.Node { return Element("em", children...) }
func Br(children ...any) *component.Node { return Element("br", children...) }
func Hr(children ...any) *component.Node { return Element("hr", children...) }

func Select(children ...any) *component.Node { return Element("select", children...) }
func Option(children ...any) *component.Node { return Element("option", children...) }
func Textarea(children ...any) *component.Node { return Element("textarea", children...) }

func Svg(children ...any) *component.Node { return Element("svg", children...) }
func Path(children ...any) *component.Node { return Element("path", children...) }
func G(children ...any) *component.Node { return Element("g", children...) }
func Circle(children ...any) *component.Node { return Element("circle", children...) }
func Rect(children ...any) *component.Node { return Element("rect", children...) }
