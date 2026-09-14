package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// UICmd returns the `goks ui` subcommand (shadcn/ui style component generator).
func UICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ui",
		Short: "Manage reusable UI components (shadcn/ui for GoKS)",
	}

	cmd.AddCommand(uiAddCmd())
	cmd.AddCommand(uiListCmd())
	return cmd
}

func uiListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all available UI components",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(color.CyanString("\n  📦 Available GoKS UI Components:"))
			fmt.Println()
			for name, desc := range componentDescriptions {
				fmt.Printf("  • %-12s %s\n", color.GreenString(name), color.HiBlackString(desc))
			}
			fmt.Printf("\n  Add a component: %s\n\n", color.YellowString("goks ui add <name>"))
		},
	}
}

func uiAddCmd() *cobra.Command {
	var appDir string

	cmd := &cobra.Command{
		Use:   "add [component...]",
		Short: "Add one or more UI components to components/ui/",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("specify at least one component to add, or 'all'. Run 'goks ui list' to see all available components")
			}

			if appDir == "" {
				appDir, _ = os.Getwd()
			}
			targetDir := filepath.Join(appDir, "components", "ui")
			if err := os.MkdirAll(targetDir, 0755); err != nil {
				return fmt.Errorf("failed to create components/ui directory: %w", err)
			}

			itemsToAdd := args
			if len(args) == 1 && strings.ToLower(args[0]) == "all" {
				itemsToAdd = make([]string, 0, len(uiTemplates))
				for name := range uiTemplates {
					itemsToAdd = append(itemsToAdd, name)
				}
			}

			fmt.Println(color.CyanString("\n  ✨ Adding GoKS UI Components:"))
			for _, name := range itemsToAdd {
				key := strings.ToLower(name)
				tmpl, ok := uiTemplates[key]
				if !ok {
					fmt.Printf("  ⚠️  Component '%s' not found. Run 'goks ui list' to see available components.\n", name)
					continue
				}

				filePath := filepath.Join(targetDir, key+".gox")
				if err := os.WriteFile(filePath, []byte(strings.TrimSpace(tmpl)+"\n"), 0644); err != nil {
					return fmt.Errorf("failed to write %s: %w", filePath, err)
				}
				fmt.Printf("  %s %s\n", color.GreenString("✓ Added"), filepath.Join("components", "ui", key+".gox"))
			}

			fmt.Println(color.HiBlackString("\n  Components are ready to use in your GOX files!"))
			return nil
		},
	}

	cmd.Flags().StringVarP(&appDir, "dir", "d", "", "Target app directory (default: current dir)")
	return cmd
}

var componentDescriptions = map[string]string{
	"button":   "Versatile button with primary, secondary, destructive, outline, and ghost variants",
	"input":    "Styled form text input with focus ring and error states",
	"card":     "Card container with Header, Title, Description, and Content sections",
	"dialog":   "Accessible modal dialog with overlay backdrop and close action",
	"badge":    "Status badge with default, success, warning, and destructive variants",
	"dropdown": "Dropdown action menu with styled items and dividers",
	"table":    "Modern data table with styled header, rows, and responsive borders",
	"image":    "Optimized responsive image component with layout shift protection and on-demand resizing",
}

var uiTemplates = map[string]string{
	"button": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type ButtonProps struct {
	Variant string // default, secondary, destructive, outline, ghost
	Size    string // sm, md, lg
	Class   string
	OnClick func()
}

type Button struct {
	component.ComponentBase
	Props    ButtonProps
	Children component.Renderable
}

func (b *Button) Render() *component.Node {
	baseClass := "inline-flex items-center justify-center font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-offset-2 rounded-md disabled:opacity-50 disabled:pointer-events-none"
	
	variantClass := "bg-blue-600 text-white hover:bg-blue-700 focus:ring-blue-500 shadow-sm"
	switch b.Props.Variant {
	case "secondary":
		variantClass = "bg-gray-100 text-gray-900 hover:bg-gray-200 focus:ring-gray-400 dark:bg-zinc-800 dark:text-zinc-100 dark:hover:bg-zinc-700"
	case "destructive":
		variantClass = "bg-red-600 text-white hover:bg-red-700 focus:ring-red-500 shadow-sm"
	case "outline":
		variantClass = "border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-zinc-200 hover:bg-gray-50 dark:hover:bg-zinc-800"
	case "ghost":
		variantClass = "text-gray-700 dark:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-800"
	}

	sizeClass := "px-4 py-2 text-sm"
	switch b.Props.Size {
	case "sm":
		sizeClass = "px-3 py-1.5 text-xs rounded"
	case "lg":
		sizeClass = "px-6 py-3 text-base rounded-lg"
	}

	finalClass := baseClass + " " + variantClass + " " + sizeClass + " " + b.Props.Class

	return (
		<button class={finalClass} onClick={b.Props.OnClick}>
			{b.Children}
		</button>
	)
}
`,

	"input": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type InputProps struct {
	Type        string // text, email, password, number
	Name        string
	Value       string
	Placeholder string
	Required    bool
	Disabled    bool
	Class       string
}

type Input struct {
	component.ComponentBase
	Props InputProps
}

func (i *Input) Render() *component.Node {
	inputType := i.Props.Type
	if inputType == "" {
		inputType = "text"
	}

	baseClass := "flex w-full rounded-md border border-gray-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-3 py-2 text-sm placeholder:text-gray-400 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent disabled:cursor-not-allowed disabled:opacity-50 transition-all duration-150"
	finalClass := baseClass + " " + i.Props.Class

	return (
		<input
			type={inputType}
			name={i.Props.Name}
			value={i.Props.Value}
			placeholder={i.Props.Placeholder}
			class={finalClass}
		/>
	)
}
`,

	"card": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type CardProps struct {
	Class string
}

type Card struct {
	component.ComponentBase
	Props    CardProps
	Children component.Renderable
}

func (c *Card) Render() *component.Node {
	finalClass := "rounded-xl border border-gray-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 shadow-sm p-6 " + c.Props.Class
	return (
		<div class={finalClass}>
			{c.Children}
		</div>
	)
}

type CardHeader struct {
	component.ComponentBase
	Children component.Renderable
}

func (c *CardHeader) Render() *component.Node {
	return (
		<div class="flex flex-col space-y-1.5 pb-4">
			{c.Children}
		</div>
	)
}

type CardTitle struct {
	component.ComponentBase
	Children component.Renderable
}

func (c *CardTitle) Render() *component.Node {
	return (
		<h3 class="text-xl font-semibold tracking-tight text-gray-900 dark:text-zinc-50">
			{c.Children}
		</h3>
	)
}
`,

	"dialog": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type DialogProps struct {
	Open    bool
	OnClose func()
	Title   string
	Class   string
}

type Dialog struct {
	component.ComponentBase
	Props    DialogProps
	Children component.Renderable
}

func (d *Dialog) Render() *component.Node {
	if !d.Props.Open {
		return component.Text("")
	}

	return (
		<div class="fixed inset-0 z-50 flex items-center justify-center">
			<div class="fixed inset-0 bg-black/50 backdrop-blur-sm transition-opacity" onClick={d.Props.OnClose}></div>
			<div class="relative z-50 w-full max-w-lg rounded-xl border border-gray-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6 shadow-xl">
				<div class="flex items-center justify-between pb-4">
					<h2 class="text-lg font-semibold text-gray-900 dark:text-zinc-50">{d.Props.Title}</h2>
					<button class="text-gray-400 hover:text-gray-500 text-sm font-medium" onClick={d.Props.OnClose}>✕</button>
				</div>
				<div class="mt-2 text-sm text-gray-600 dark:text-zinc-400">
					{d.Children}
				</div>
			</div>
		</div>
	)
}
`,

	"badge": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type BadgeProps struct {
	Variant string // default, success, warning, destructive, outline
	Class   string
}

type Badge struct {
	component.ComponentBase
	Props    BadgeProps
	Children component.Renderable
}

func (b *Badge) Render() *component.Node {
	base := "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors"
	variant := "bg-blue-100 text-blue-800 dark:bg-blue-950 dark:text-blue-300"

	switch b.Props.Variant {
	case "success":
		variant = "bg-green-100 text-green-800 dark:bg-green-950 dark:text-green-300"
	case "warning":
		variant = "bg-yellow-100 text-yellow-800 dark:bg-yellow-950 dark:text-yellow-300"
	case "destructive":
		variant = "bg-red-100 text-red-800 dark:bg-red-950 dark:text-red-300"
	case "outline":
		variant = "border border-gray-300 dark:border-zinc-700 text-gray-700 dark:text-zinc-300"
	}

	finalClass := base + " " + variant + " " + b.Props.Class
	return (
		<span class={finalClass}>
			{b.Children}
		</span>
	)
}
`,

	"dropdown": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type DropdownProps struct {
	Open  bool
	Class string
}

type Dropdown struct {
	component.ComponentBase
	Props    DropdownProps
	Children component.Renderable
}

func (d *Dropdown) Render() *component.Node {
	if !d.Props.Open {
		return component.Text("")
	}
	finalClass := "absolute right-0 mt-2 w-48 rounded-md border border-gray-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 py-1 shadow-lg ring-1 ring-black ring-opacity-5 z-40 " + d.Props.Class
	return (
		<div class={finalClass}>
			{d.Children}
		</div>
	)
}

type DropdownItemProps struct {
	OnClick func()
	Class   string
}

type DropdownItem struct {
	component.ComponentBase
	Props    DropdownItemProps
	Children component.Renderable
}

func (i *DropdownItem) Render() *component.Node {
	finalClass := "block w-full px-4 py-2 text-left text-sm text-gray-700 dark:text-zinc-200 hover:bg-gray-100 dark:hover:bg-zinc-800 cursor-pointer " + i.Props.Class
	return (
		<button class={finalClass} onClick={i.Props.OnClick}>
			{i.Children}
		</button>
	)
}
`,

	"table": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/html"
)

type TableProps struct {
	Class string
}

type Table struct {
	component.ComponentBase
	Props    TableProps
	Children component.Renderable
}

func (t *Table) Render() *component.Node {
	finalClass := "w-full text-left text-sm text-gray-700 dark:text-zinc-300 border-collapse " + t.Props.Class
	return (
		<div class="relative w-full overflow-auto rounded-lg border border-gray-200 dark:border-zinc-800">
			<table class={finalClass}>
				{t.Children}
			</table>
		</div>
	)
}

type TableRow struct {
	component.ComponentBase
	Children component.Renderable
}

func (r *TableRow) Render() *component.Node {
	return (
		<tr class="border-b border-gray-200 dark:border-zinc-800 transition-colors hover:bg-gray-50/50 dark:hover:bg-zinc-800/50">
			{r.Children}
		</tr>
	)
}
`,
	"image": `package ui

import (
	"github.com/misbakhul29/goks/pkg/component"
	"github.com/misbakhul29/goks/pkg/image"
)

type ImageProps struct {
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

type Image struct {
	component.ComponentBase
	Props ImageProps
}

func (img *Image) Render() *component.Node {
	return image.New(image.Props{
		Src:      img.Props.Src,
		Alt:      img.Props.Alt,
		Width:    img.Props.Width,
		Height:   img.Props.Height,
		Quality:  img.Props.Quality,
		Priority: img.Props.Priority,
		Class:    img.Props.Class,
		Sizes:    img.Props.Sizes,
		Style:    img.Props.Style,
	}).Render()
}
`,
}
