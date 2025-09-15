package editor

import (
	"io"

	"github.com/unidoc/unioffice/v2/color"
	"github.com/unidoc/unioffice/v2/common"
	"github.com/unidoc/unioffice/v2/document"
	"github.com/unidoc/unioffice/v2/measurement"
	"github.com/unidoc/unioffice/v2/schema/soo/wml"
)

// Aliases for unioffice types to simplify usage
type (
	UnderlineStyle  = wml.ST_Underline
	HighlightColor  = wml.ST_HighlightColor
	MergeV          = wml.ST_Merge
	HdrFtr          = wml.ST_HdrFtr
	StyleType       = wml.ST_StyleType
	Alignment       = wml.ST_Jc
	LineSpacingRule = wml.ST_LineSpacingRule
	Distance        = measurement.Distance
	OfficeImage     = common.Image
	ImageRef        = common.ImageRef
	TOCOptions      = document.TOCOptions
	OfficeColor     = color.Color
)

// WordDocument wraps a unioffice document.
type WordDocument struct {
	doc *document.Document
}

// Paragraph wraps a unioffice paragraph.
type Paragraph struct {
	p document.Paragraph
}

// Run wraps a unioffice run.
type Run struct {
	r document.Run
}

// RunProperties wraps unioffice run properties.
type RunProperties struct {
	props document.RunProperties
}

// Table wraps a unioffice table.
type Table struct {
	t document.Table
}

// Row wraps a unioffice row.
type Row struct {
	r document.Row
}

// Cell wraps a unioffice cell.
type Cell struct {
	c document.Cell
}

// CellProperties wraps unioffice cell properties.
type CellProperties struct {
	props document.CellProperties
}

// CellBorders wraps unioffice cell borders.
type CellBorders struct {
	borders document.CellBorders
}

// Header wraps a unioffice header.
type Header struct {
	h document.Header
}

// Footer wraps a unioffice footer.
type Footer struct {
	f document.Footer
}

// Section wraps a unioffice section.
type Section struct {
	s document.Section
}

// Styles wraps unioffice styles.
type Styles struct {
	s document.Styles
}

// Style wraps a unioffice style.
type Style struct {
	s document.Style
}

// ParagraphProperties wraps unioffice paragraph properties.
type ParagraphProperties struct {
	props document.ParagraphProperties
}

// Bookmark wraps a unioffice bookmark.
type Bookmark struct {
	b document.Bookmark
}

type Hyperlink struct {
	h document.HyperLink
}

// init initializes the unioffice license.
func init() {
	// It is recommended to set a metered license key.
	// See https://cloud.unidoc.io for more details.
	// err := license.SetMeteredKey("YOUR_LICENSE_KEY")
	// if err != nil {
	// 	panic(err)
	// }
}

// --- 1. File Operations ---

// NewWordDocument creates a new blank Word document.
func NewWordDocument() *WordDocument {
	return &WordDocument{doc: document.New()}
}

// OpenWordDocument opens an existing .docx file.
func OpenWordDocument(path string) (*WordDocument, error) {
	doc, err := document.Open(path)
	if err != nil {
		return nil, err
	}
	return &WordDocument{doc: doc}, nil
}

// SaveToFile saves the document to the specified path.
func (w *WordDocument) SaveToFile(path string) error {
	return w.doc.SaveToFile(path)
}

// Write writes the document content to an io.Writer.
func (w *WordDocument) Write(writer io.Writer) error {
	return w.doc.Save(writer)
}

// Copy creates a deep copy of the document.
func (w *WordDocument) Copy() (*WordDocument, error) {
	newDoc, err := w.doc.Copy()
	if err != nil {
		return nil, err
	}
	return &WordDocument{doc: newDoc}, nil
}

// Append appends the content of another document to the current one.
func (w *WordDocument) Append(doc2 *WordDocument) error {
	return w.doc.Append(doc2.doc)
}

// --- 2. Content Addition & Editing ---

// AddParagraph adds a new paragraph to the end of the document.
func (w *WordDocument) AddParagraph() Paragraph {
	return Paragraph{p: w.doc.AddParagraph()}
}

// AddTable adds a new table to the document.
func (w *WordDocument) AddTable() Table {
	return Table{t: w.doc.AddTable()}
}

// AddImage adds an image to the document.
func (w *WordDocument) AddImage(img OfficeImage) (ImageRef, error) {
	return w.doc.AddImage(img)
}

// AddBookmark adds a bookmark to the document.
func (w *WordDocument) AddBookmark(name string) Bookmark {
	// 需要通过段落来添加书签
	para := w.doc.AddParagraph()
	bookmark := para.AddBookmark(name)
	return Bookmark{b: bookmark}
}

// AddTOC adds a table of contents to a run.
func (r Run) AddTOC(options *TOCOptions) {
	r.r.AddTOC(options)
}

// MailMerge performs a mail merge operation.
func (w *WordDocument) MailMerge(data map[string]string) {
	w.doc.MailMerge(data)
}

// --- 3. Paragraph and Run ---

// AddRun adds a new run to a paragraph.
func (p Paragraph) AddRun() Run {
	return Run{r: p.p.AddRun()}
}

// AddHyperlink adds a hyperlink to a run.
func (p Paragraph) AddHyperlink(url string) Hyperlink {
	hyperlink := p.p.AddHyperLink()
	hyperlink.SetTarget(url)
	return Hyperlink{h: hyperlink}
}

// SetText sets the text content of a run by clearing existing content first.
func (r Run) SetText(text string) {
	r.r.ClearContent()
	r.r.AddText(text)
}

// AddText appends text to a run.
func (r Run) AddText(text string) {
	r.r.AddText(text)
}

// AddTab adds a tab to a run.
func (r Run) AddTab() {
	r.r.AddTab()
}

// AddPageBreak adds a page break to a run.
func (r Run) AddPageBreak() {
	r.r.AddPageBreak()
}

// Properties returns the properties of a run.
func (r Run) Properties() RunProperties {
	return RunProperties{props: r.r.Properties()}
}

// SetBold sets the bold property.
func (rp RunProperties) SetBold(bold bool) {
	rp.props.SetBold(bold)
}

// SetItalic sets the italic property.
func (rp RunProperties) SetItalic(italic bool) {
	rp.props.SetItalic(italic)
}

// SetUnderline sets the underline property.
func (rp RunProperties) SetUnderline(style UnderlineStyle, clr OfficeColor) {
	rp.props.SetUnderline(style, clr)
}

// SetFontSize sets the font size.
func (rp RunProperties) SetFontSize(size Distance) {
	rp.props.SetSize(size)
}

// SetColor sets the font color.
func (rp RunProperties) SetColor(clr OfficeColor) {
	rp.props.SetColor(clr)
}

// SetHighlight sets the highlight color.
func (rp RunProperties) SetHighlight(clr HighlightColor) {
	rp.props.SetHighlight(clr)
}

// --- 4. Table ---

// AddRow adds a new row to a table.
func (t Table) AddRow() Row {
	return Row{r: t.t.AddRow()}
}

// AddCell adds a new cell to a row.
func (r Row) AddCell() Cell {
	return Cell{c: r.r.AddCell()}
}

// AddParagraph adds a paragraph to a cell.
func (c Cell) AddParagraph() Paragraph {
	return Paragraph{p: c.c.AddParagraph()}
}

// SetText is a shortcut to set text in a cell.
func (c Cell) SetText(text string) {
	c.c.AddParagraph().AddRun().AddText(text)
}

// Properties returns the properties of a cell.
func (c Cell) Properties() CellProperties {
	return CellProperties{props: c.c.Properties()}
}

// SetWidth sets the width of a cell.
func (cp CellProperties) SetWidth(width Distance) {
	cp.props.SetWidth(width)
}

// SetVerticalMerge sets vertical merging for a cell.
func (cp CellProperties) SetVerticalMerge(mergeVal MergeV) {
	cp.props.SetVerticalMerge(mergeVal)
}

// SetColumnSpan sets the number of grid columns spanned by a cell.
func (cp CellProperties) SetColumnSpan(cols int) {
	cp.props.SetColumnSpan(cols)
}

// Borders returns the border configuration for a cell.
func (cp CellProperties) Borders() CellBorders {
	return CellBorders{borders: cp.props.Borders()}
}

// --- 5. Header and Footer ---

// AddHeader adds a header to the document.
func (w *WordDocument) AddHeader() Header {
	return Header{h: w.doc.AddHeader()}
}

// AddFooter adds a footer to the document.
func (w *WordDocument) AddFooter() Footer {
	return Footer{f: w.doc.AddFooter()}
}

// SetHeader sets a header for a document section.
func (s Section) SetHeader(h Header, t HdrFtr) {
	s.s.SetHeader(h.h, t)
}

// AddParagraph adds a paragraph to a header.
func (h Header) AddParagraph() Paragraph {
	return Paragraph{p: h.h.AddParagraph()}
}

// AddParagraph adds a paragraph to a footer.
func (f Footer) AddParagraph() Paragraph {
	return Paragraph{p: f.f.AddParagraph()}
}

// --- 6. Styles and Formatting ---

// Styles returns the document's style collection.
func (w *WordDocument) Styles() Styles {
	return Styles{s: w.doc.Styles}
}

// AddStyle adds a new style to the document.
func (s Styles) AddStyle(name string, styleType StyleType, isDefault bool) Style {
	return Style{s: s.s.AddStyle(name, styleType, isDefault)}
}

// SetStyle applies a style to a paragraph.
func (p Paragraph) SetStyle(name string) {
	p.p.SetStyle(name)
}

// Properties returns the properties of a paragraph.
func (p Paragraph) Properties() ParagraphProperties {
	return ParagraphProperties{props: p.p.Properties()}
}

// SetAlignment sets the alignment of a paragraph.
func (pp ParagraphProperties) SetAlignment(align Alignment) {
	pp.props.SetAlignment(align)
}

// SetLeftIndent sets the left indent of a paragraph.
func (p Paragraph) SetLeftIndent(indent Distance) {
	p.p.SetLeftIndent(indent)
}

// SetLineSpacing sets the line spacing of a paragraph.
func (p Paragraph) SetLineSpacing(spacing Distance, rule LineSpacingRule) {
	p.p.SetLineSpacing(spacing, rule)
}
