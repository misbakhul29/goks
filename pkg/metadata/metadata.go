package metadata

import (
	"fmt"
	"html"
	"strings"

	"github.com/misbakhul29/goks/pkg/component"
)

// MetadataProvider can be implemented by any Component (such as Layout or Page)
// to provide SEO and document head metadata.
type MetadataProvider interface {
	Metadata() Metadata
}

// Metadata defines the complete SEO and HTML document metadata.
type Metadata struct {
	Title         string
	TitleTemplate string // e.g. "%s | GoKS" (applied when child provides Title)
	AbsoluteTitle string // ignores any TitleTemplate from parent
	Description   string
	Keywords      []string
	Authors       []Author
	Creator       string
	Publisher     string
	Robots        string // e.g. "index, follow" or "noindex, nofollow"
	ThemeColor    string
	Canonical     string
	Manifest      string
	Charset       string            // defaults to "UTF-8"
	Viewport      string            // defaults to "width=device-width, initial-scale=1.0"
	Icons         *Icons
	OpenGraph     *OpenGraph
	Twitter       *Twitter
	Languages     map[string]string // hreflang alternates (e.g. "en-US": "/en", "id-ID": "/id")
	Other         map[string]string // custom <meta name="..." content="...">
}

// Author defines content author information.
type Author struct {
	Name string
	URL  string
}

// Icons defines favicon and touch icon paths.
type Icons struct {
	Icon     string // standard favicon (e.g. "/favicon.ico")
	Shortcut string // shortcut icon
	Apple    string // apple-touch-icon (e.g. "/apple-touch-icon.png")
}

// OpenGraph defines Open Graph social sharing metadata.
type OpenGraph struct {
	Title       string
	Description string
	URL         string
	SiteName    string
	Images      []OGImage
	Locale      string
	Type        string // defaults to "website"
}

// OGImage defines an Open Graph image.
type OGImage struct {
	URL    string
	Width  int
	Height int
	Alt    string
	Type   string
}

// Twitter defines Twitter Card social sharing metadata.
type Twitter struct {
	Card        string // "summary", "summary_large_image", "app", "player"
	Title       string
	Description string
	Site        string // e.g. "@goksframework"
	Creator     string // e.g. "@author"
	Images      []string
}

// Merge combines parent and child metadata in hierarchical order.
// Child values take precedence, and TitleTemplate from parent is applied to child Title.
func Merge(parent, child Metadata) Metadata {
	result := parent

	// Title resolution
	if child.AbsoluteTitle != "" {
		result.Title = child.AbsoluteTitle
		result.AbsoluteTitle = child.AbsoluteTitle
	} else if child.Title != "" {
		if parent.TitleTemplate != "" && strings.Contains(parent.TitleTemplate, "%s") {
			result.Title = fmt.Sprintf(parent.TitleTemplate, child.Title)
		} else {
			result.Title = child.Title
		}
	}
	if child.TitleTemplate != "" {
		result.TitleTemplate = child.TitleTemplate
	}

	if child.Description != "" {
		result.Description = child.Description
	}
	if len(child.Keywords) > 0 {
		result.Keywords = child.Keywords
	}
	if len(child.Authors) > 0 {
		result.Authors = child.Authors
	}
	if child.Creator != "" {
		result.Creator = child.Creator
	}
	if child.Publisher != "" {
		result.Publisher = child.Publisher
	}
	if child.Robots != "" {
		result.Robots = child.Robots
	}
	if child.ThemeColor != "" {
		result.ThemeColor = child.ThemeColor
	}
	if child.Canonical != "" {
		result.Canonical = child.Canonical
	}
	if child.Manifest != "" {
		result.Manifest = child.Manifest
	}
	if child.Charset != "" {
		result.Charset = child.Charset
	}
	if child.Viewport != "" {
		result.Viewport = child.Viewport
	}

	// Icons
	if child.Icons != nil {
		if result.Icons == nil {
			result.Icons = &Icons{}
		}
		if child.Icons.Icon != "" {
			result.Icons.Icon = child.Icons.Icon
		}
		if child.Icons.Shortcut != "" {
			result.Icons.Shortcut = child.Icons.Shortcut
		}
		if child.Icons.Apple != "" {
			result.Icons.Apple = child.Icons.Apple
		}
	}

	// OpenGraph
	if child.OpenGraph != nil {
		if result.OpenGraph == nil {
			result.OpenGraph = &OpenGraph{}
		}
		if child.OpenGraph.Title != "" {
			result.OpenGraph.Title = child.OpenGraph.Title
		}
		if child.OpenGraph.Description != "" {
			result.OpenGraph.Description = child.OpenGraph.Description
		}
		if child.OpenGraph.URL != "" {
			result.OpenGraph.URL = child.OpenGraph.URL
		}
		if child.OpenGraph.SiteName != "" {
			result.OpenGraph.SiteName = child.OpenGraph.SiteName
		}
		if child.OpenGraph.Locale != "" {
			result.OpenGraph.Locale = child.OpenGraph.Locale
		}
		if child.OpenGraph.Type != "" {
			result.OpenGraph.Type = child.OpenGraph.Type
		}
		if len(child.OpenGraph.Images) > 0 {
			result.OpenGraph.Images = child.OpenGraph.Images
		}
	}

	// Twitter
	if child.Twitter != nil {
		if result.Twitter == nil {
			result.Twitter = &Twitter{}
		}
		if child.Twitter.Card != "" {
			result.Twitter.Card = child.Twitter.Card
		}
		if child.Twitter.Title != "" {
			result.Twitter.Title = child.Twitter.Title
		}
		if child.Twitter.Description != "" {
			result.Twitter.Description = child.Twitter.Description
		}
		if child.Twitter.Site != "" {
			result.Twitter.Site = child.Twitter.Site
		}
		if child.Twitter.Creator != "" {
			result.Twitter.Creator = child.Twitter.Creator
		}
		if len(child.Twitter.Images) > 0 {
			result.Twitter.Images = child.Twitter.Images
		}
	}

	// Languages / Alternates
	if len(child.Languages) > 0 {
		if result.Languages == nil {
			result.Languages = make(map[string]string)
		}
		for k, v := range child.Languages {
			result.Languages[k] = v
		}
	}

	// Other meta tags
	if len(child.Other) > 0 {
		if result.Other == nil {
			result.Other = make(map[string]string)
		}
		for k, v := range child.Other {
			result.Other[k] = v
		}
	}

	return result
}

// RenderHTML converts the metadata into an HTML string containing all appropriate <title>,
// <meta>, and <link> tags.
func RenderHTML(m Metadata) string {
	var sb strings.Builder

	charset := m.Charset
	if charset == "" {
		charset = "UTF-8"
	}
	sb.WriteString(fmt.Sprintf("  <meta charset=\"%s\" />\n", html.EscapeString(charset)))

	viewport := m.Viewport
	if viewport == "" {
		viewport = "width=device-width, initial-scale=1.0"
	}
	sb.WriteString(fmt.Sprintf("  <meta name=\"viewport\" content=\"%s\" />\n", html.EscapeString(viewport)))

	if m.Title != "" {
		sb.WriteString(fmt.Sprintf("  <title>%s</title>\n", html.EscapeString(m.Title)))
	}

	if m.Description != "" {
		sb.WriteString(fmt.Sprintf("  <meta name=\"description\" content=\"%s\" />\n", html.EscapeString(m.Description)))
	}

	if len(m.Keywords) > 0 {
		sb.WriteString(fmt.Sprintf("  <meta name=\"keywords\" content=\"%s\" />\n", html.EscapeString(strings.Join(m.Keywords, ", "))))
	}

	for _, a := range m.Authors {
		if a.Name != "" {
			sb.WriteString(fmt.Sprintf("  <meta name=\"author\" content=\"%s\" />\n", html.EscapeString(a.Name)))
		}
	}

	if m.Creator != "" {
		sb.WriteString(fmt.Sprintf("  <meta name=\"creator\" content=\"%s\" />\n", html.EscapeString(m.Creator)))
	}

	if m.Publisher != "" {
		sb.WriteString(fmt.Sprintf("  <meta name=\"publisher\" content=\"%s\" />\n", html.EscapeString(m.Publisher)))
	}

	if m.Robots != "" {
		sb.WriteString(fmt.Sprintf("  <meta name=\"robots\" content=\"%s\" />\n", html.EscapeString(m.Robots)))
	}

	if m.ThemeColor != "" {
		sb.WriteString(fmt.Sprintf("  <meta name=\"theme-color\" content=\"%s\" />\n", html.EscapeString(m.ThemeColor)))
	}

	if m.Canonical != "" {
		sb.WriteString(fmt.Sprintf("  <link rel=\"canonical\" href=\"%s\" />\n", html.EscapeString(m.Canonical)))
	}

	if m.Manifest != "" {
		sb.WriteString(fmt.Sprintf("  <link rel=\"manifest\" href=\"%s\" />\n", html.EscapeString(m.Manifest)))
	}

	// Icons
	if m.Icons != nil {
		if m.Icons.Icon != "" {
			sb.WriteString(fmt.Sprintf("  <link rel=\"icon\" href=\"%s\" />\n", html.EscapeString(m.Icons.Icon)))
		}
		if m.Icons.Shortcut != "" {
			sb.WriteString(fmt.Sprintf("  <link rel=\"shortcut icon\" href=\"%s\" />\n", html.EscapeString(m.Icons.Shortcut)))
		}
		if m.Icons.Apple != "" {
			sb.WriteString(fmt.Sprintf("  <link rel=\"apple-touch-icon\" href=\"%s\" />\n", html.EscapeString(m.Icons.Apple)))
		}
	}

	// Languages
	for lang, href := range m.Languages {
		sb.WriteString(fmt.Sprintf("  <link rel=\"alternate\" hreflang=\"%s\" href=\"%s\" />\n", html.EscapeString(lang), html.EscapeString(href)))
	}

	// OpenGraph
	if m.OpenGraph != nil {
		og := m.OpenGraph
		ogTitle := og.Title
		if ogTitle == "" {
			ogTitle = m.Title
		}
		if ogTitle != "" {
			sb.WriteString(fmt.Sprintf("  <meta property=\"og:title\" content=\"%s\" />\n", html.EscapeString(ogTitle)))
		}

		ogDesc := og.Description
		if ogDesc == "" {
			ogDesc = m.Description
		}
		if ogDesc != "" {
			sb.WriteString(fmt.Sprintf("  <meta property=\"og:description\" content=\"%s\" />\n", html.EscapeString(ogDesc)))
		}

		if og.URL != "" {
			sb.WriteString(fmt.Sprintf("  <meta property=\"og:url\" content=\"%s\" />\n", html.EscapeString(og.URL)))
		}
		if og.SiteName != "" {
			sb.WriteString(fmt.Sprintf("  <meta property=\"og:site_name\" content=\"%s\" />\n", html.EscapeString(og.SiteName)))
		}
		if og.Locale != "" {
			sb.WriteString(fmt.Sprintf("  <meta property=\"og:locale\" content=\"%s\" />\n", html.EscapeString(og.Locale)))
		}

		ogType := og.Type
		if ogType == "" {
			ogType = "website"
		}
		sb.WriteString(fmt.Sprintf("  <meta property=\"og:type\" content=\"%s\" />\n", html.EscapeString(ogType)))

		for _, img := range og.Images {
			if img.URL != "" {
				sb.WriteString(fmt.Sprintf("  <meta property=\"og:image\" content=\"%s\" />\n", html.EscapeString(img.URL)))
				if img.Width > 0 {
					sb.WriteString(fmt.Sprintf("  <meta property=\"og:image:width\" content=\"%d\" />\n", img.Width))
				}
				if img.Height > 0 {
					sb.WriteString(fmt.Sprintf("  <meta property=\"og:image:height\" content=\"%d\" />\n", img.Height))
				}
				if img.Alt != "" {
					sb.WriteString(fmt.Sprintf("  <meta property=\"og:image:alt\" content=\"%s\" />\n", html.EscapeString(img.Alt)))
				}
				if img.Type != "" {
					sb.WriteString(fmt.Sprintf("  <meta property=\"og:image:type\" content=\"%s\" />\n", html.EscapeString(img.Type)))
				}
			}
		}
	}

	// Twitter
	if m.Twitter != nil {
		tw := m.Twitter
		card := tw.Card
		if card == "" {
			card = "summary_large_image"
		}
		sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:card\" content=\"%s\" />\n", html.EscapeString(card)))

		twTitle := tw.Title
		if twTitle == "" {
			twTitle = m.Title
		}
		if twTitle != "" {
			sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:title\" content=\"%s\" />\n", html.EscapeString(twTitle)))
		}

		twDesc := tw.Description
		if twDesc == "" {
			twDesc = m.Description
		}
		if twDesc != "" {
			sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:description\" content=\"%s\" />\n", html.EscapeString(twDesc)))
		}

		if tw.Site != "" {
			sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:site\" content=\"%s\" />\n", html.EscapeString(tw.Site)))
		}
		if tw.Creator != "" {
			sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:creator\" content=\"%s\" />\n", html.EscapeString(tw.Creator)))
		}
		for _, img := range tw.Images {
			if img != "" {
				sb.WriteString(fmt.Sprintf("  <meta name=\"twitter:image\" content=\"%s\" />\n", html.EscapeString(img)))
			}
		}
	}

	// Custom other tags
	for name, val := range m.Other {
		if strings.HasPrefix(name, "og:") {
			sb.WriteString(fmt.Sprintf("  <meta property=\"%s\" content=\"%s\" />\n", html.EscapeString(name), html.EscapeString(val)))
		} else {
			sb.WriteString(fmt.Sprintf("  <meta name=\"%s\" content=\"%s\" />\n", html.EscapeString(name), html.EscapeString(val)))
		}
	}

	return sb.String()
}

// ExtractFromTree traverses the component/element node tree and extracts/merges metadata
// from any component implementing MetadataProvider.
func ExtractFromTree(node *component.Node) Metadata {
	var collected []Metadata
	collectMetadata(node, &collected)

	if len(collected) == 0 {
		return Metadata{}
	}

	result := collected[0]
	for i := 1; i < len(collected); i++ {
		result = Merge(result, collected[i])
	}
	return result
}

func collectMetadata(node *component.Node, out *[]Metadata) {
	if node == nil {
		return
	}

	if node.Type == component.NodeTypeComponent && node.Component != nil {
		if mp, ok := node.Component.(MetadataProvider); ok {
			*out = append(*out, mp.Metadata())
		}
		// Render component to inspect deeper
		childNode := node.Component.Render()
		collectMetadata(childNode, out)
		return
	}

	for _, child := range node.Children {
		collectMetadata(child, out)
	}
}
