package google

// This file contains factory functions for all popular Google Fonts.
// Each function accepts Options and returns a *Font that is automatically
// registered in the global font registry.
//
// Usage:
//
//	var inter = google.Inter(google.Options{
//	    Variable: "--font-inter",
//	    Subsets:  []string{"latin"},
//	    Weight:   []string{"400", "500", "600", "700"},
//	})

// ── Sans-Serif ────────────────────────────────────────────────────────────────

// Inter loads the Inter font family.
func Inter(opts Options) *Font {
	return newFont("Inter", "Inter", opts)
}

// Roboto loads the Roboto font family.
func Roboto(opts Options) *Font {
	return newFont("Roboto", "Roboto", opts)
}

// OpenSans loads the Open Sans font family.
func OpenSans(opts Options) *Font {
	return newFont("Open Sans", "Open+Sans", opts)
}

// Lato loads the Lato font family.
func Lato(opts Options) *Font {
	return newFont("Lato", "Lato", opts)
}

// Montserrat loads the Montserrat font family.
func Montserrat(opts Options) *Font {
	return newFont("Montserrat", "Montserrat", opts)
}

// Poppins loads the Poppins font family.
func Poppins(opts Options) *Font {
	return newFont("Poppins", "Poppins", opts)
}

// Nunito loads the Nunito font family.
func Nunito(opts Options) *Font {
	return newFont("Nunito", "Nunito", opts)
}

// NunitoSans loads the Nunito Sans font family.
func NunitoSans(opts Options) *Font {
	return newFont("Nunito Sans", "Nunito+Sans", opts)
}

// Raleway loads the Raleway font family.
func Raleway(opts Options) *Font {
	return newFont("Raleway", "Raleway", opts)
}

// Oswald loads the Oswald font family.
func Oswald(opts Options) *Font {
	return newFont("Oswald", "Oswald", opts)
}

// Ubuntu loads the Ubuntu font family.
func Ubuntu(opts Options) *Font {
	return newFont("Ubuntu", "Ubuntu", opts)
}

// Mulish loads the Mulish font family.
func Mulish(opts Options) *Font {
	return newFont("Mulish", "Mulish", opts)
}

// WorkSans loads the Work Sans font family.
func WorkSans(opts Options) *Font {
	return newFont("Work Sans", "Work+Sans", opts)
}

// Outfit loads the Outfit font family.
func Outfit(opts Options) *Font {
	return newFont("Outfit", "Outfit", opts)
}

// PlusJakartaSans loads the Plus Jakarta Sans font family.
func PlusJakartaSans(opts Options) *Font {
	return newFont("Plus Jakarta Sans", "Plus+Jakarta+Sans", opts)
}

// DmSans loads the DM Sans font family.
func DmSans(opts Options) *Font {
	return newFont("DM Sans", "DM+Sans", opts)
}

// Figtree loads the Figtree font family.
func Figtree(opts Options) *Font {
	return newFont("Figtree", "Figtree", opts)
}

// Manrope loads the Manrope font family.
func Manrope(opts Options) *Font {
	return newFont("Manrope", "Manrope", opts)
}

// Rubik loads the Rubik font family.
func Rubik(opts Options) *Font {
	return newFont("Rubik", "Rubik", opts)
}

// Karla loads the Karla font family.
func Karla(opts Options) *Font {
	return newFont("Karla", "Karla", opts)
}

// Jost loads the Jost font family.
func Jost(opts Options) *Font {
	return newFont("Jost", "Jost", opts)
}

// Sora loads the Sora font family.
func Sora(opts Options) *Font {
	return newFont("Sora", "Sora", opts)
}

// SpaceGrotesk loads the Space Grotesk font family.
func SpaceGrotesk(opts Options) *Font {
	return newFont("Space Grotesk", "Space+Grotesk", opts)
}

// Lexend loads the Lexend font family.
func Lexend(opts Options) *Font {
	return newFont("Lexend", "Lexend", opts)
}

// Onest loads the Onest font family.
func Onest(opts Options) *Font {
	return newFont("Onest", "Onest", opts)
}

// ── Serif ─────────────────────────────────────────────────────────────────────

// Merriweather loads the Merriweather font family.
func Merriweather(opts Options) *Font {
	return newFont("Merriweather", "Merriweather", opts)
}

// PlayfairDisplay loads the Playfair Display font family.
func PlayfairDisplay(opts Options) *Font {
	return newFont("Playfair Display", "Playfair+Display", opts)
}

// Lora loads the Lora font family.
func Lora(opts Options) *Font {
	return newFont("Lora", "Lora", opts)
}

// CrimsonText loads the Crimson Text font family.
func CrimsonText(opts Options) *Font {
	return newFont("Crimson Text", "Crimson+Text", opts)
}

// LibreBaskerville loads the Libre Baskerville font family.
func LibreBaskerville(opts Options) *Font {
	return newFont("Libre Baskerville", "Libre+Baskerville", opts)
}

// Fraunces loads the Fraunces font family.
func Fraunces(opts Options) *Font {
	return newFont("Fraunces", "Fraunces", opts)
}

// ── Monospace ─────────────────────────────────────────────────────────────────

// JetBrainsMono loads the JetBrains Mono font family.
func JetBrainsMono(opts Options) *Font {
	return newFont("JetBrains Mono", "JetBrains+Mono", opts)
}

// FiraCode loads the Fira Code font family.
func FiraCode(opts Options) *Font {
	return newFont("Fira Code", "Fira+Code", opts)
}

// SourceCodePro loads the Source Code Pro font family.
func SourceCodePro(opts Options) *Font {
	return newFont("Source Code Pro", "Source+Code+Pro", opts)
}

// Inconsolata loads the Inconsolata font family.
func Inconsolata(opts Options) *Font {
	return newFont("Inconsolata", "Inconsolata", opts)
}

// SpaceMono loads the Space Mono font family.
func SpaceMono(opts Options) *Font {
	return newFont("Space Mono", "Space+Mono", opts)
}

// Geist loads the Geist font family.
func Geist(opts Options) *Font {
	return newFont("Geist", "Geist", opts)
}

// GeistMono loads the Geist Mono font family.
func GeistMono(opts Options) *Font {
	return newFont("Geist Mono", "Geist+Mono", opts)
}

// ── Display / Decorative ──────────────────────────────────────────────────────

// PermanentMarker loads the Permanent Marker font family.
func PermanentMarker(opts Options) *Font {
	return newFont("Permanent Marker", "Permanent+Marker", opts)
}

// RubikGlitch loads the Rubik Glitch font family.
func RubikGlitch(opts Options) *Font {
	return newFont("Rubik Glitch", "Rubik+Glitch", opts)
}

// Pacifico loads the Pacifico font family.
func Pacifico(opts Options) *Font {
	return newFont("Pacifico", "Pacifico", opts)
}

// Lobster loads the Lobster font family.
func Lobster(opts Options) *Font {
	return newFont("Lobster", "Lobster", opts)
}

// Righteous loads the Righteous font family.
func Righteous(opts Options) *Font {
	return newFont("Righteous", "Righteous", opts)
}

// Bungee loads the Bungee font family.
func Bungee(opts Options) *Font {
	return newFont("Bungee", "Bungee", opts)
}

// BungeeShade loads the Bungee Shade font family.
func BungeeShade(opts Options) *Font {
	return newFont("Bungee Shade", "Bungee+Shade", opts)
}

// Bangers loads the Bangers font family.
func Bangers(opts Options) *Font {
	return newFont("Bangers", "Bangers", opts)
}

// Audiowide loads the Audiowide font family.
func Audiowide(opts Options) *Font {
	return newFont("Audiowide", "Audiowide", opts)
}

// Exo2 loads the Exo 2 font family.
func Exo2(opts Options) *Font {
	return newFont("Exo 2", "Exo+2", opts)
}

// Orbitron loads the Orbitron font family.
func Orbitron(opts Options) *Font {
	return newFont("Orbitron", "Orbitron", opts)
}

// Raleway loads the Teko font family.
func Teko(opts Options) *Font {
	return newFont("Teko", "Teko", opts)
}

// ── Variable-weight fonts ──────────────────────────────────────────────────────

// Anybody loads the Anybody variable font family.
func Anybody(opts Options) *Font {
	return newFont("Anybody", "Anybody", opts)
}

// Barlow loads the Barlow font family.
func Barlow(opts Options) *Font {
	return newFont("Barlow", "Barlow", opts)
}

// BarlowCondensed loads the Barlow Condensed font family.
func BarlowCondensed(opts Options) *Font {
	return newFont("Barlow Condensed", "Barlow+Condensed", opts)
}

// Cabin loads the Cabin font family.
func Cabin(opts Options) *Font {
	return newFont("Cabin", "Cabin", opts)
}

// Quicksand loads the Quicksand font family.
func Quicksand(opts Options) *Font {
	return newFont("Quicksand", "Quicksand", opts)
}

// Varela loads the Varela font family.
func Varela(opts Options) *Font {
	return newFont("Varela", "Varela", opts)
}

// VarelaRound loads the Varela Round font family.
func VarelaRound(opts Options) *Font {
	return newFont("Varela Round", "Varela+Round", opts)
}
