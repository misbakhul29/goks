// Package google provides Google Fonts integration for GoKS, similar to next/font/google in Next.js.
//
// Usage:
//
//	var spaceGrotesk = google.SpaceGrotesk(google.Options{
//	    Variable: "--font-space-grotesk",
//	    Subsets:  []string{"latin"},
//	})
//
//	// In your layout body:
//	html.Body(component.Props{"class": spaceGrotesk.Variable()}, children...)
//
// Fonts are automatically registered in a global registry. The GoKS server
// reads this registry and injects the appropriate <link> preconnect and <style>
// tags into the document <head> on every page.
package google

import (
	"fmt"
	"strings"
	"sync"
)

// Classes joins the CSS variable names of the given fonts and any extra class
// strings into a single space-separated string, ready to use in a GOX class attribute.
//
// Usage (equivalent to Next.js template literals):
//
//	<body class={google.Classes(geistSans, fontGlitch, "antialiased")}>
//
// Each *Font argument contributes its CSS variable name (e.g. "--font-geist-sans").
// String arguments are passed through as-is (e.g. "antialiased", "dark").
func Classes(parts ...any) string {
	var classes []string
	for _, p := range parts {
		switch v := p.(type) {
		case *Font:
			if v != nil {
				classes = append(classes, v.variable)
			}
		case string:
			if v != "" {
				classes = append(classes, v)
			}
		}
	}
	return strings.Join(classes, " ")
}


// Options configures how a Google Font is loaded.
type Options struct {
	// Variable is the CSS custom property name that will hold the font-family value.
	// e.g. "--font-space-grotesk"
	// If empty, a default name is derived from the font name.
	Variable string

	// Subsets is a list of Unicode subsets to load.
	// e.g. []string{"latin"}, []string{"latin", "latin-ext"}
	// Defaults to []string{"latin"} if empty.
	Subsets []string

	// Weight is a list of font weights to include.
	// e.g. []string{"400"}, []string{"400", "700"}, []string{"100..900"} for variable fonts.
	// Defaults to []string{"400"} if empty.
	Weight []string

	// Style is the font style ("normal", "italic").
	// Defaults to "normal" if empty.
	Style string

	// Display controls the font-display CSS property.
	// e.g. "swap" (recommended), "block", "fallback", "optional".
	// Defaults to "swap" if empty.
	Display string
}

// Font represents a loaded Google Font with its configuration.
// Create instances using the factory functions (e.g., google.Inter, google.SpaceGrotesk).
type Font struct {
	name     string // Google Font name, e.g. "Space Grotesk"
	familyID string // URL-safe font family ID, e.g. "Space+Grotesk"
	variable string // CSS variable name, e.g. "--font-space-grotesk"
	class    string // CSS class name, e.g. "font-space-grotesk"
	subsets  []string
	weights  []string
	style    string
	display  string
}

// Variable returns the CSS custom property name (e.g., "--font-space-grotesk").
// Use this in your component's class prop to apply the font variable:
//
//	html.Body(component.Props{"class": myFont.Variable()}, ...)
func (f *Font) Variable() string {
	return f.variable
}

// ClassName returns the CSS class name generated for this font
// (e.g., "font-space-grotesk"). This class sets the CSS variable on the element.
func (f *Font) ClassName() string {
	return f.class
}

// HeadHTML returns the HTML string to inject into <head> for this font.
// It includes <link rel="preconnect"> tags and a <style> block that declares
// the CSS custom property.
func (f *Font) HeadHTML() string {
	var sb strings.Builder

	// Build Google Fonts URL
	googleFontsURL := f.buildGoogleFontsURL()

	// Preconnect hints for performance
	sb.WriteString(`  <link rel="preconnect" href="https://fonts.googleapis.com" />` + "\n")
	sb.WriteString(`  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />` + "\n")

	// Load the font stylesheet
	sb.WriteString(fmt.Sprintf(`  <link rel="stylesheet" href="%s" />`, googleFontsURL))
	sb.WriteString("\n")

	// Declare the CSS variable and class
	display := f.display
	if display == "" {
		display = "swap"
	}
	sb.WriteString("  <style>\n")
	sb.WriteString(fmt.Sprintf("    .%s {\n", f.class))
	sb.WriteString(fmt.Sprintf("      %s: '%s', sans-serif;\n", f.variable, f.name))
	sb.WriteString("    }\n")
	sb.WriteString("  </style>\n")

	return sb.String()
}

// buildGoogleFontsURL constructs the Google Fonts API v2 URL.
func (f *Font) buildGoogleFontsURL() string {
	display := f.display
	if display == "" {
		display = "swap"
	}

	weights := f.weights
	if len(weights) == 0 {
		weights = []string{"400"}
	}

	style := f.style
	if style == "" {
		style = "normal"
	}

	// Build the ital,wght axis if needed
	var axisTag string
	var axisValues []string

	if style == "italic" {
		axisTag = "ital,wght@"
		for _, w := range weights {
			axisValues = append(axisValues, "0,"+w)
			axisValues = append(axisValues, "1,"+w)
		}
	} else {
		axisTag = "wght@"
		axisValues = weights
	}

	familyParam := fmt.Sprintf("family=%s:%s%s",
		f.familyID,
		axisTag,
		strings.Join(axisValues, ";"),
	)

	// Subsets are not a direct query param in v2, but we include display
	return fmt.Sprintf("https://fonts.googleapis.com/css2?%s&display=%s", familyParam, display)
}

// --- Global Font Registry ---

var (
	registryMu sync.RWMutex
	registry   []*Font
	// Track which family IDs are already registered to avoid duplicates.
	registered = map[string]bool{}
)

// register adds a font to the global registry if not already present.
func register(f *Font) *Font {
	registryMu.Lock()
	defer registryMu.Unlock()

	key := f.variable + "|" + strings.Join(f.weights, ",") + "|" + f.style
	if !registered[key] {
		registered[key] = true
		registry = append(registry, f)
	}
	return f
}

// RegisteredFonts returns all fonts that have been registered in the global registry.
// This is called by the GoKS server runtime to inject font HTML into the document head.
func RegisteredFonts() []*Font {
	registryMu.RLock()
	defer registryMu.RUnlock()

	out := make([]*Font, len(registry))
	copy(out, registry)
	return out
}

// AllFontsHeadHTML returns a combined HTML string of all registered fonts' head HTML.
// This is a convenience function used by the server to inject everything at once.
func AllFontsHeadHTML() string {
	fonts := RegisteredFonts()
	if len(fonts) == 0 {
		return ""
	}

	// Deduplicate the preconnect hints — only emit once for all fonts.
	var sb strings.Builder
	sb.WriteString(`  <link rel="preconnect" href="https://fonts.googleapis.com" />` + "\n")
	sb.WriteString(`  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />` + "\n")

	// Emit each font's <link> and <style> (skip the preconnect since we already wrote it)
	for _, f := range fonts {
		googleFontsURL := f.buildGoogleFontsURL()
		sb.WriteString(fmt.Sprintf(`  <link rel="stylesheet" href="%s" />`, googleFontsURL) + "\n")
		sb.WriteString("  <style>\n")
		sb.WriteString(fmt.Sprintf("    .%s {\n", f.class))
		sb.WriteString(fmt.Sprintf("      %s: '%s', sans-serif;\n", f.variable, f.name))
		sb.WriteString("    }\n")
		sb.WriteString("  </style>\n")
	}

	return sb.String()
}

// newFont is the internal constructor used by all font factory functions.
func newFont(name, familyID string, opts Options) *Font {
	// Derive CSS variable name if not provided
	variable := opts.Variable
	if variable == "" {
		slug := strings.ToLower(strings.ReplaceAll(name, " ", "-"))
		variable = "--font-" + slug
	}

	// Derive CSS class from variable name (strip the leading "--")
	class := strings.TrimPrefix(variable, "--")

	subsets := opts.Subsets
	if len(subsets) == 0 {
		subsets = []string{"latin"}
	}

	weights := opts.Weight
	if len(weights) == 0 {
		weights = []string{"400"}
	}

	style := opts.Style
	if style == "" {
		style = "normal"
	}

	display := opts.Display
	if display == "" {
		display = "swap"
	}

	f := &Font{
		name:     name,
		familyID: familyID,
		variable: variable,
		class:    class,
		subsets:  subsets,
		weights:  weights,
		style:    style,
		display:  display,
	}

	return register(f)
}
