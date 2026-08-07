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

	// Inline code is AMBER ON THE PAGE, with no background of its own.
	//
	// It used to carry the flap background, and that caused two visible faults once
	// code BLOCKS gained the same background. Glamour pads a wrapped line with whatever
	// style was last active, so a line ending in `cost[i]` got a dark rectangle floating
	// at its right margin. Worse, a prose line that BEGAN with an inline span was
	// indistinguishable from a block — same background, same position — so the whole
	// line was filled and a bar of flap ran across the middle of a paragraph.
	//
	// Amber alone still reads as code, which is the job. The background now means
	// exactly one thing: this line is a block.
	Code: ansi.StyleBlock{
		StylePrimitive: ansi.StylePrimitive{
			Color:  p(cAmber),
			Prefix: " ",
			Suffix: " ",
		},
	},
	// An example block is DATA, not prose — the input, the output, and the walk-through
	// are what you check your understanding against, and they were rendering in a grey a
	// shade off body text with no background at all. On a wall of paragraphs that reads
	// as more paragraph.
	//
	// The flap background is the same treatment inline `code` already carries, so a
	// block and a span of code look related instead of unrelated. It is one step off the
	// page base — enough to bound the block, not enough to compete with the statement.
	CodeBlock: ansi.StyleCodeBlock{
		StyleBlock: ansi.StyleBlock{
			StylePrimitive: ansi.StylePrimitive{
				Color:           p(cBone),
				BackgroundColor: p(cFlap),
			},
			// No margin. The background now bounds the block, so an indent adds
			// nothing visually — and it pushed a full-width line one cell over the
			// pane, where its padding wrapped onto a blank row of its own.
			Margin: p(uint(0)),
		},
		Chroma: &ansi.Chroma{
			// Every Chroma token needs the background too. Without it a highlighted
			// keyword punches a hole in the block.
			Text:          ansi.StylePrimitive{Color: p(cBone), BackgroundColor: p(cFlap)},
			Keyword:       ansi.StylePrimitive{Color: p(cAmber), Bold: p(true), BackgroundColor: p(cFlap)},
			KeywordType:   ansi.StylePrimitive{Color: p(cAmber), BackgroundColor: p(cFlap)},
			NameFunction:  ansi.StylePrimitive{Color: p(cBone), Bold: p(true), BackgroundColor: p(cFlap)},
			NameBuiltin:   ansi.StylePrimitive{Color: p(cAmber), BackgroundColor: p(cFlap)},
			Comment:       ansi.StylePrimitive{Color: p(cDim), Italic: p(true), BackgroundColor: p(cFlap)},
			LiteralString: ansi.StylePrimitive{Color: p(cDim), BackgroundColor: p(cFlap)},
			LiteralNumber: ansi.StylePrimitive{Color: p(cBone), BackgroundColor: p(cFlap)},
			Operator:      ansi.StylePrimitive{Color: p(cDim), BackgroundColor: p(cFlap)},
			Punctuation:   ansi.StylePrimitive{Color: p(cDim), BackgroundColor: p(cFlap)},
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
	// Glamour colours the characters of a code block and pads the rest of the line with
	// plain spaces, so the background stops at the last character. Fill it out, or the
	// examples read as a ragged smear rather than a bounded block.
	return paintCodeBlocks(out, width), nil
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
