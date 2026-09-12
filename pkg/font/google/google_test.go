package google_test

import (
	"strings"
	"testing"

	"github.com/misbakhul29/goks/pkg/font/google"
)

func TestFont_Variable(t *testing.T) {
	f := google.SpaceGrotesk(google.Options{
		Variable: "--font-space-grotesk",
		Subsets:  []string{"latin"},
	})
	if got := f.Variable(); got != "--font-space-grotesk" {
		t.Errorf("Variable() = %q, want %q", got, "--font-space-grotesk")
	}
}

func TestFont_Variable_Default(t *testing.T) {
	// When no Variable is provided, it should be auto-derived from font name.
	f := google.Inter(google.Options{
		Subsets: []string{"latin"},
	})
	if got := f.Variable(); got != "--font-inter" {
		t.Errorf("Variable() (default) = %q, want %q", got, "--font-inter")
	}
}

func TestFont_ClassName(t *testing.T) {
	f := google.Rubik(google.Options{
		Variable: "--font-rubik",
	})
	if got := f.ClassName(); got != "font-rubik" {
		t.Errorf("ClassName() = %q, want %q", got, "font-rubik")
	}
}

func TestFont_HeadHTML_ContainsPreconnect(t *testing.T) {
	f := google.Montserrat(google.Options{
		Variable: "--font-montserrat",
		Subsets:  []string{"latin"},
	})
	html := f.HeadHTML()
	if !strings.Contains(html, `fonts.googleapis.com`) {
		t.Error("HeadHTML() should contain preconnect to fonts.googleapis.com")
	}
	if !strings.Contains(html, `fonts.gstatic.com`) {
		t.Error("HeadHTML() should contain preconnect to fonts.gstatic.com")
	}
}

func TestFont_HeadHTML_ContainsFontLink(t *testing.T) {
	f := google.Poppins(google.Options{
		Variable: "--font-poppins",
		Weight:   []string{"400", "700"},
		Subsets:  []string{"latin"},
	})
	html := f.HeadHTML()
	if !strings.Contains(html, "Poppins") {
		t.Errorf("HeadHTML() should contain font family name 'Poppins', got:\n%s", html)
	}
	if !strings.Contains(html, `rel="stylesheet"`) {
		t.Errorf("HeadHTML() should contain a stylesheet link, got:\n%s", html)
	}
}

func TestFont_HeadHTML_ContainsCSSVariable(t *testing.T) {
	f := google.RubikGlitch(google.Options{
		Variable: "--font-rubik-glitch",
		Weight:   []string{"400"},
	})
	html := f.HeadHTML()
	if !strings.Contains(html, "--font-rubik-glitch") {
		t.Errorf("HeadHTML() should declare the CSS variable '--font-rubik-glitch', got:\n%s", html)
	}
}

func TestFont_HeadHTML_ContainsStyleTag(t *testing.T) {
	f := google.PermanentMarker(google.Options{
		Variable: "--font-permanent-marker",
		Weight:   []string{"400"},
	})
	html := f.HeadHTML()
	if !strings.Contains(html, "<style>") {
		t.Errorf("HeadHTML() should contain a <style> block, got:\n%s", html)
	}
}

func TestFont_Display_Default(t *testing.T) {
	f := google.Lato(google.Options{
		Variable: "--font-lato-default-display",
	})
	html := f.HeadHTML()
	// Default display should be "swap"
	if !strings.Contains(html, "display=swap") {
		t.Errorf("HeadHTML() should use display=swap by default, got:\n%s", html)
	}
}

func TestFont_Display_Custom(t *testing.T) {
	f := google.Geist(google.Options{
		Variable: "--font-geist-optional",
		Display:  "optional",
	})
	html := f.HeadHTML()
	if !strings.Contains(html, "display=optional") {
		t.Errorf("HeadHTML() should use display=optional, got:\n%s", html)
	}
}

func TestAllFontsHeadHTML_DeduplicatesPreconnect(t *testing.T) {
	// Even with multiple fonts registered, preconnect should appear at most once.
	allHTML := google.AllFontsHeadHTML()
	count := strings.Count(allHTML, `rel="preconnect" href="https://fonts.googleapis.com"`)
	if count != 1 {
		t.Errorf("AllFontsHeadHTML() should have exactly 1 googleapis preconnect, got %d", count)
	}
}

func TestAllFontsHeadHTML_ContainsAllFonts(t *testing.T) {
	allHTML := google.AllFontsHeadHTML()
	// Both Space+Grotesk and Rubik+Glitch should appear (registered by earlier tests)
	if !strings.Contains(allHTML, "Space+Grotesk") {
		t.Error("AllFontsHeadHTML() should contain Space+Grotesk")
	}
	if !strings.Contains(allHTML, "Rubik+Glitch") {
		t.Error("AllFontsHeadHTML() should contain Rubik+Glitch")
	}
}
