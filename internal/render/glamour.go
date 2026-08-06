package render

import (
	"fmt"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/glamour/ansi"
)

// Departure Board palette, duplicated here as plain strings because Glamour's style
// config takes *string, not lipgloss.Color.
//
// These MUST stay in sync with internal/tui/theme. If you change a token there, change
// it here. (They cannot share a constant: theme's values are lipgloss.Color and this
// package must not import the TUI.)
const (
	cInk   = "#0F1014"
	cFlap  = "#1B1D24"
	cHinge = "#2A2D36"
	cAmber = "#E8A33D"
	cBone  = "#E6E3DA"
	cDim   = "#6B7080"
)

func p[T any](v T) *T { return &v }

// style is the Glamour theme for problem statements and editorials.
//
// The discipline from docs/DESIGN.md holds inside rendered content too: amber is the
// system's voice (headings, emphasis, rules), bone is content, dim is metadata. No
// green, no red — those belong to the judge alone, and an editorial is not a verdict.
var style = ansi.StyleConfig{
	Document: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cBone)},
		Margin:         p(uint(0)),
	},
	BlockQuote: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cDim), Italic: p(true)},
		Indent:         p(uint(1)),
		IndentToken:    p("│ "),
	},
	Paragraph: ansi.StyleBlock{},
	List: ansi.StyleList{
		StyleBlock:  ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{}},
		LevelIndent: 2,
	},
	Heading: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cAmber), Bold: p(true)},
		Margin:         p(uint(0)),
	},
	H1: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cAmber), Bold: p(true), Prefix: "", Suffix: ""},
	},
	H2: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cAmber), Bold: p(true), Prefix: ""},
	},
	H3: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{Color: p(cAmber), Prefix: ""},
	},
	Text:   ansi.StylePrimitive{},
	Strong: ansi.StylePrimitive{Color: p(cBone), Bold: p(true)},
	Emph:   ansi.StylePrimitive{Color: p(cBone), Italic: p(true)},
	HorizontalRule: ansi.StylePrimitive{
		Color:  p(cHinge),
		Format: "\n────────────────────────────\n",
	},
	Item:        ansi.StylePrimitive{BlockPrefix: "• "},
	Enumeration: ansi.StylePrimitive{BlockPrefix: ". ", Color: p(cAmber)},

	// Links are dim: in a terminal they are not clickable, so making them loud would
	// give them a prominence they cannot cash.
	Link:     ansi.StylePrimitive{Color: p(cDim), Underline: p(true)},
	LinkText: ansi.StylePrimitive{Color: p(cBone)},

	// Image markers are already bracketed by the HTML converter; this only colors them.
	Image:     ansi.StylePrimitive{Color: p(cAmber)},
	ImageText: ansi.StylePrimitive{Color: p(cAmber), Format: "{{.text}}"},

	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:           p(cAmber),
			BackgroundColor: p(cFlap),
			Prefix:          " ",
			Suffix:          " ",
		},
	},
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{Color: p(cBone)},
			Margin:         p(uint(1)),
		},
		Chroma: &ansi.Chroma{
			Text:          ansi.StylePrimitive{Color: p(cBone)},
			Keyword:       ansi.StylePrimitive{Color: p(cAmber), Bold: p(true)},
			KeywordType:   ansi.StylePrimitive{Color: p(cAmber)},
			NameFunction:  ansi.StylePrimitive{Color: p(cBone), Bold: p(true)},
			NameBuiltin:   ansi.StylePrimitive{Color: p(cAmber)},
			Comment:       ansi.StylePrimitive{Color: p(cDim), Italic: p(true)},
			LiteralString: ansi.StylePrimitive{Color: p(cDim)},
			LiteralNumber: ansi.StylePrimitive{Color: p(cBone)},
			Operator:      ansi.StylePrimitive{Color: p(cDim)},
			Punctuation:   ansi.StylePrimitive{Color: p(cDim)},
			Background:    ansi.StylePrimitive{BackgroundColor: p(cInk)},
		},
	},
	Table: ansi.StyleTable{
		StyleBlock: ansi.StyleBlock{StylePrimitive: ansi.StylePrimitive{}},
	},
	DefinitionDescription: ansi.StylePrimitive{BlockPrefix: "\n  "},
}

// renderers are cached per width. Building a TermRenderer is not cheap, and the detail
// pane re-renders on every resize and every cursor move.
var (
	mu        sync.Mutex
	renderers = map[int]*glamour.TermRenderer{}
)

func rendererFor(width int) (*glamour.TermRenderer, error) {
	if width < 20 {
		width = 20
	}
	mu.Lock()
	defer mu.Unlock()

	if r, ok := renderers[width]; ok {
		return r, nil
	}
	r, err := glamour.NewTermRenderer(
		glamour.WithStyles(style),
		glamour.WithWordWrap(width),
		glamour.WithEmoji(),
	)
	if err != nil {
		return nil, fmt.Errorf("build renderer: %w", err)
	}
	renderers[width] = r
	return r, nil
}

// Markdown renders markdown to styled terminal output at the given width.
func Markdown(md string, width int) (string, error) {
	r, err := rendererFor(width)
	if err != nil {
		return "", err
	}
	out, err := r.Render(md)
	if err != nil {
		return "", fmt.Errorf("render markdown: %w", err)
	}
	return out, nil
}

// HTML converts LeetCode HTML and renders it in one step, returning the referenced
// images so the caller can wire up "open image" on the bracket markers.
func HTML(input string, width int) (string, []Image, error) {
	doc, err := HTMLToMarkdown(input)
	if err != nil {
		return "", nil, err
	}
	out, err := Markdown(doc.Markdown, width)
	if err != nil {
		return "", doc.Images, err
	}
	return out, doc.Images, nil
}
