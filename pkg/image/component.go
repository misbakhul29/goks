package image

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/misbakhul29/goks/pkg/component"
)

// Props defines the configuration for the Image component.
type Props struct {
	Src      string
	Alt      string
	Width    int
	Height   int
	Quality  int
	Priority bool
	Class    string
	Sizes    string
	Style    string
}

// Image represents a responsive, optimized image component with layout shift protection.
type Image struct {
	component.ComponentBase
	Props Props
}

// New creates a new Image component.
func New(props Props) *Image {
	if props.Quality <= 0 {
		props.Quality = 75
	}
	return &Image{Props: props}
}

// URL generates the optimized image endpoint URL.
func URL(src string, width, quality int) string {
	if quality <= 0 {
		quality = 75
	}
	return fmt.Sprintf("/__goks_image?url=%s&w=%d&q=%d", url.QueryEscape(src), width, quality)
}

// SrcSet generates a responsive srcset string.
func SrcSet(src string, quality int, widths ...int) string {
	if len(widths) == 0 {
		widths = []int{640, 750, 828, 1080, 1200, 1920}
	}
	var parts []string
	for _, w := range widths {
		parts = append(parts, fmt.Sprintf("%s %dw", URL(src, w, quality), w))
	}
	return strings.Join(parts, ", ")
}

// Render outputs the HTML <img> node with responsive attributes and layout shift protection.
func (img *Image) Render() *component.Node {
	p := img.Props
	if p.Quality <= 0 {
		p.Quality = 75
	}

	optW := p.Width
	if optW <= 0 {
		optW = 1080
	}

	optSrc := URL(p.Src, optW, p.Quality)
	srcSet := SrcSet(p.Src, p.Quality)

	loading := "lazy"
	decoding := "async"
	fetchPriority := "auto"
	if p.Priority {
		loading = "eager"
		fetchPriority = "high"
	}

	aspectRatioStyle := ""
	if p.Width > 0 && p.Height > 0 {
		aspectRatioStyle = fmt.Sprintf("aspect-ratio: %d / %d; ", p.Width, p.Height)
	}

	combinedStyle := aspectRatioStyle + p.Style

	attrs := component.Props{
		"src":           optSrc,
		"srcset":        srcSet,
		"alt":           p.Alt,
		"loading":       loading,
		"decoding":      decoding,
		"fetchpriority": fetchPriority,
	}

	if p.Width > 0 {
		attrs["width"] = p.Width
	}
	if p.Height > 0 {
		attrs["height"] = p.Height
	}
	if p.Class != "" {
		attrs["class"] = p.Class
	}
	if p.Sizes != "" {
		attrs["sizes"] = p.Sizes
	} else {
		attrs["sizes"] = "100vw"
	}
	if combinedStyle != "" {
		attrs["style"] = strings.TrimSpace(combinedStyle)
	}

	return component.H("img", attrs)
}
