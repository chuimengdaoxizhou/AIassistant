
package editor

import (
	"github.com/unidoc/unioffice/v2/presentation"
)

// PowerPointPresentation wraps a unioffice presentation.
type PowerPointPresentation struct {
	ppt *presentation.Presentation
}

// Slide wraps a unioffice slide.
type PPTXSlide struct {
	slide presentation.Slide
}

// SlideLayout wraps a unioffice slide layout.
type SlideLayout struct {
	layout presentation.SlideLayout
}

// TextBox wraps a unioffice text box.
type TextBox struct {
	tb presentation.TextBox
}

// Image wraps a unioffice image on a slide.
type PPTXImage struct {
	img presentation.Image
}

// PlaceHolder wraps a unioffice placeholder.
type PlaceHolder struct {
	ph presentation.PlaceHolder
}

// --- 1. File Operations ---

// NewPowerPointPresentation creates a new blank PowerPoint presentation.
func NewPowerPointPresentation() *PowerPointPresentation {
	return &PowerPointPresentation{ppt: presentation.New()}
}

// OpenPowerPointPresentation opens an existing .pptx file.
func OpenPowerPointPresentation(path string) (*PowerPointPresentation, error) {
	ppt, err := presentation.Open(path)
	if err != nil {
		return nil, err
	}
	return &PowerPointPresentation{ppt: ppt}, nil
}

// SaveToFile saves the presentation to the specified path.
func (p *PowerPointPresentation) SaveToFile(path string) error {
	return p.ppt.SaveToFile(path)
}

// --- 2. Slide Management ---

// AddSlide adds a new blank slide to the presentation.
func (p *PowerPointPresentation) AddSlide() PPTXSlide {
	return PPTXSlide{slide: p.ppt.AddSlide()}
}

// AddSlideWithLayout adds a new slide based on a specified layout.
func (p *PowerPointPresentation) AddSlideWithLayout(layout SlideLayout) (PPTXSlide, error) {
	s, err := p.ppt.AddSlideWithLayout(layout.layout)
	if err != nil {
		return PPTXSlide{}, err
	}
	return PPTXSlide{slide: s}, nil
}

// Slides returns all slides in the presentation.
func (p *PowerPointPresentation) Slides() []PPTXSlide {
	var slides []PPTXSlide
	for _, s := range p.ppt.Slides() {
		slides = append(slides, PPTXSlide{slide: s})
	}
	return slides
}

// GetLayoutByName returns a slide layout by its name.
func (p *PowerPointPresentation) GetLayoutByName(name string) (SlideLayout, error) {
	l, err := p.ppt.GetLayoutByName(name)
	if err != nil {
		return SlideLayout{}, err
	}
	return SlideLayout{layout: l}, nil
}

// --- 3. Content and Shapes ---

// AddTextBox adds a new text box to a slide.
func (s PPTXSlide) AddTextBox() TextBox {
	return TextBox{tb: s.slide.AddTextBox()}
}

// AddImage adds an image to a slide.
func (s PPTXSlide) AddImage(imgRef ImageRef) PPTXImage {
	return PPTXImage{img: s.slide.AddImage(imgRef)}
}

// PlaceHolders returns all placeholders on a slide.
func (s PPTXSlide) PlaceHolders() []PlaceHolder {
	var phs []PlaceHolder
	for _, p := range s.slide.PlaceHolders() {
		phs = append(phs, PlaceHolder{ph: p})
	}
	return phs
}

// SetText sets the text content of a text box.
func (t TextBox) SetText(text string) {
	// unioffice TextBox does not have a direct SetText, we add a paragraph and a run.
	p := t.tb.AddParagraph()
	r := p.AddRun()
	r.SetText(text)
}

// SetText sets the text content of a placeholder.
func (p PlaceHolder) SetText(text string) {
	p.ph.SetText(text)
}
