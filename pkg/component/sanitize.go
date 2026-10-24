package component

import (
	"reflect"
	"regexp"
	"strings"
	"unicode"
)

var validAttrName = regexp.MustCompile(`^[a-zA-Z_:][-a-zA-Z0-9_:.]*$`)

var commonEvents = map[string]bool{
	"onclick": true, "ondblclick": true, "onmousedown": true, "onmouseup": true,
	"onmouseover": true, "onmouseout": true, "onmousemove": true, "onmouseenter": true,
	"onmouseleave": true, "onkeydown": true, "onkeyup": true, "onkeypress": true,
	"onchange": true, "oninput": true, "onsubmit": true, "onreset": true,
	"onfocus": true, "onblur": true, "onscroll": true, "onload": true, "onerror": true,
	"ontouchstart": true, "ontouchend": true, "ontouchmove": true, "ontouchcancel": true,
	"onanimationstart": true, "onanimationend": true, "ontransitionend": true,
}

// IsValidAttributeName validates an HTML/XML attribute name against safe characters (CWE-79).
func IsValidAttributeName(name string) bool {
	return validAttrName.MatchString(name)
}

// isFunc returns true if val is a function.
func isFunc(val any) bool {
	if val == nil {
		return false
	}
	return reflect.TypeOf(val).Kind() == reflect.Func
}

// isEventProp returns true if the key represents an event listener.
func isEventProp(k string, v any) bool {
	if len(k) < 3 {
		return false
	}
	lower := strings.ToLower(k)
	if !strings.HasPrefix(lower, "on") {
		return false
	}
	if lower == "once" || lower == "one" || lower == "only" {
		return false
	}
	// If 3rd character is uppercase (e.g. onClick, OnChange, ONCLICK)
	if unicode.IsUpper(rune(k[2])) {
		return true
	}
	// If it matches common DOM events
	if commonEvents[lower] {
		return true
	}
	// If the value is a Go function
	return isFunc(v)
}

// isDangerousURLScheme checks if an attribute value uses a dangerous scheme (javascript:, vbscript:, or data: on links).
func isDangerousURLScheme(attrName, val string) bool {
	lowerAttr := strings.ToLower(attrName)
	if lowerAttr != "href" && lowerAttr != "src" && lowerAttr != "action" && lowerAttr != "formaction" {
		return false
	}

	// Remove ASCII whitespace and control characters (which browsers strip before scheme parsing)
	var sb strings.Builder
	for _, r := range val {
		if r > ' ' && r != '\t' && r != '\n' && r != '\r' {
			sb.WriteRune(r)
		}
	}
	cleaned := strings.ToLower(sb.String())

	if strings.HasPrefix(cleaned, "javascript:") || strings.HasPrefix(cleaned, "vbscript:") {
		return true
	}
	if (lowerAttr == "href" || lowerAttr == "action" || lowerAttr == "formaction") && strings.HasPrefix(cleaned, "data:") {
		return true
	}
	return false
}
