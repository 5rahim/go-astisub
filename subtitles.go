package astisub

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/asticode/go-astikit"
	"golang.org/x/net/html"
)

// Bytes
var (
	BytesBOM           = []byte{239, 187, 191}
	bytesLineSeparator = []byte("\n")
	bytesSpace         = []byte(" ")
)

// Colors
var (
	ColorBlack   = &Color{}
	ColorBlue    = &Color{Blue: 255}
	ColorCyan    = &Color{Blue: 255, Green: 255}
	ColorGray    = &Color{Blue: 128, Green: 128, Red: 128}
	ColorGreen   = &Color{Green: 128}
	ColorLime    = &Color{Green: 255}
	ColorMagenta = &Color{Blue: 255, Red: 255}
	ColorMaroon  = &Color{Red: 128}
	ColorNavy    = &Color{Blue: 128}
	ColorOlive   = &Color{Green: 128, Red: 128}
	ColorPurple  = &Color{Blue: 128, Red: 128}
	ColorRed     = &Color{Red: 255}
	ColorSilver  = &Color{Blue: 192, Green: 192, Red: 192}
	ColorTeal    = &Color{Blue: 128, Green: 128}
	ColorYellow  = &Color{Green: 255, Red: 255}
	ColorWhite   = &Color{Blue: 255, Green: 255, Red: 255}
)

// Errors
var (
	ErrInvalidExtension   = errors.New("astisub: invalid extension")
	ErrNoSubtitlesToWrite = errors.New("astisub: no subtitles to write")
)

// HTML Escape
var (
	htmlEscaper   = strings.NewReplacer("&", "&amp;", "<", "&lt;", "\u00A0", "&nbsp;")
	htmlUnescaper = strings.NewReplacer("&amp;", "&", "&lt;", "<", "&nbsp;", "\u00A0")
)

// Now allows testing functions using it
var Now = func() time.Time {
	return time.Now()
}

// Options represents open or write options
type Options struct {
	Filename string
	Teletext TeletextOptions
	STL      STLOptions
}

// Open opens a subtitle reader based on options
func Open(o Options) (s *Subtitles, err error) {
	// Open the file
	var f *os.File
	if f, err = os.Open(o.Filename); err != nil {
		err = fmt.Errorf("astisub: opening %s failed: %w", o.Filename, err)
		return
	}
	defer f.Close()

	// Parse the content
	switch filepath.Ext(strings.ToLower(o.Filename)) {
	case ".srt":
		s, err = ReadFromSRT(f)
	case ".ssa", ".ass":
		s, err = ReadFromSSA(f)
	case ".stl":
		s, err = ReadFromSTL(f, o.STL)
	case ".ts":
		s, err = ReadFromTeletext(f, o.Teletext)
	case ".ttml":
		s, err = ReadFromTTML(f)
	case ".vtt":
		s, err = ReadFromWebVTT(f)
	default:
		err = ErrInvalidExtension
	}
	return
}

// OpenFile opens a file regardless of other options
func OpenFile(filename string) (*Subtitles, error) {
	return Open(Options{Filename: filename})
}

// Subtitles represents an ordered list of items with formatting
type Subtitles struct {
	Items    []*Item
	Metadata *Metadata
	Regions  map[string]*Region
	Styles   map[string]*Style
}

// NewSubtitles creates new subtitles
func NewSubtitles() *Subtitles {
	return &Subtitles{
		Metadata: &Metadata{},
		Regions:  make(map[string]*Region),
		Styles:   make(map[string]*Style),
	}
}

// Item represents a text to show between 2 time boundaries with formatting
type Item struct {
	Comments    []string
	Index       int
	EndAt       time.Duration
	InlineStyle *StyleAttributes
	Lines       []Line
	Region      *Region
	StartAt     time.Duration
	Style       *Style
}

// String implements the Stringer interface
func (i Item) String() string {
	var os []string
	for _, l := range i.Lines {
		os = append(os, l.String())
	}
	return strings.Join(os, " - ")
}

// Color represents a color
type Color struct {
	Alpha, Blue, Green, Red uint8
}

// newColorFromSSAString builds a new color based on an SSA string
func newColorFromSSAString(s string, base int) (c *Color, err error) {
	var i int64
	if i, err = strconv.ParseInt(s, base, 64); err != nil {
		err = fmt.Errorf("parsing int %s with base %d failed: %w", s, base, err)
		return
	}
	c = &Color{
		Alpha: uint8(i>>24) & 0xff,
		Blue:  uint8(i>>16) & 0xff,
		Green: uint8(i>>8) & 0xff,
		Red:   uint8(i) & 0xff,
	}
	return
}

// SSAString expresses the color as an SSA string
func (c *Color) SSAString() string {
	return fmt.Sprintf("%.8x", uint32(c.Alpha)<<24|uint32(c.Blue)<<16|uint32(c.Green)<<8|uint32(c.Red))
}

// TTMLString expresses the color as a TTML string
func (c *Color) TTMLString() string {
	return fmt.Sprintf("%.6x", uint32(c.Red)<<16|uint32(c.Green)<<8|uint32(c.Blue))
}

type Justification int

var (
	JustificationUnchanged = Justification(1)
	JustificationLeft      = Justification(2)
	JustificationCentered  = Justification(3)
	JustificationRight     = Justification(4)
)

// StyleAttributes represents style attributes
type StyleAttributes struct {
	SRTBold              bool
	SRTColor             *string
	SRTItalics           bool
	SRTPosition          byte // 1-9 numpad layout
	SRTUnderline         bool
	SSAAlignment         *int
	SSAAlphaLevel        *float64
	SSAAngle             *float64 // degrees
	SSABackColour        *Color
	SSABold              *bool
	SSABorderStyle       *int
	SSAEffect            string
	SSAEncoding          *int
	SSAFontName          string
	SSAFontSize          *float64
	SSAItalic            *bool
	SSALayer             *int
	SSAMarginLeft        *int // pixels
	SSAMarginRight       *int // pixels
	SSAMarginVertical    *int // pixels
	SSAMarked            *bool
	SSAOutline           *float64 // pixels
	SSAOutlineColour     *Color
	SSAPrimaryColour     *Color
	SSAScaleX            *float64 // %
	SSAScaleY            *float64 // %
	SSASecondaryColour   *Color
	SSAShadow            *float64 // pixels
	SSASpacing           *float64 // pixels
	SSAStrikeout         *bool
	SSAUnderline         *bool
	STLBoxing            *bool
	STLItalics           *bool
	STLJustification     *Justification
	STLPosition          *STLPosition
	STLUnderline         *bool
	TeletextColor        *Color
	TeletextDoubleHeight *bool
	TeletextDoubleSize   *bool
	TeletextDoubleWidth  *bool
	TeletextSpacesAfter  *int
	TeletextSpacesBefore *int
	// TODO Use pointers with real types below
	TTMLBackgroundColor  *string // https://htmlcolorcodes.com/fr/
	TTMLColor            *string
	TTMLDirection        *string
	TTMLDisplay          *string
	TTMLDisplayAlign     *string
	TTMLExtent           *string
	TTMLFontFamily       *string
	TTMLFontSize         *string
	TTMLFontStyle        *string
	TTMLFontWeight       *string
	TTMLLineHeight       *string
	TTMLOpacity          *string
	TTMLOrigin           *string
	TTMLOverflow         *string
	TTMLPadding          *string
	TTMLShowBackground   *string
	TTMLTextAlign        *string
	TTMLTextDecoration   *string
	TTMLTextOutline      *string
	TTMLUnicodeBidi      *string
	TTMLVisibility       *string
	TTMLWrapOption       *string
	TTMLWritingMode      *string
	TTMLZIndex           *int
	WebVTTAlign          string
	WebVTTBold           bool
	WebVTTItalics        bool
	WebVTTLine           string
	WebVTTLines          int
	WebVTTPosition       string
	WebVTTRegionAnchor   string
	WebVTTScroll         string
	WebVTTSize           string
	WebVTTStyles         []string
	WebVTTTags           []WebVTTTag
	WebVTTUnderline      bool
	WebVTTVertical       string
	WebVTTViewportAnchor string
	WebVTTWidth          string
}

type WebVTTTag struct {
	Name       string
	Annotation string
	Classes    []string
}

func (t WebVTTTag) startTag() string {
	if t.Name == "" {
		return ""
	}

	s := t.Name
	if len(t.Classes) > 0 {
		s += "." + strings.Join(t.Classes, ".")
	}

	if t.Annotation != "" {
		s += " " + t.Annotation
	}

	return "<" + s + ">"
}

func (t WebVTTTag) endTag() string {
	if t.Name == "" {
		return ""
	}
	return "</" + t.Name + ">"
}

func (sa *StyleAttributes) propagateSRTAttributes() {
	// copy relevant attrs to WebVTT ones
	if sa.SRTColor != nil {
		// TODO: handle non-default colors that need custom styles
		sa.TTMLColor = sa.SRTColor
	}

	switch sa.SRTPosition {
	case 7: // top-left
		sa.WebVTTAlign = "left"
		sa.WebVTTPosition = "10%"
	case 8: // top-center
		sa.WebVTTPosition = "10%"
	case 9: // top-right
		sa.WebVTTAlign = "right"
		sa.WebVTTPosition = "10%"
	case 4: // middle-left
		sa.WebVTTAlign = "left"
		sa.WebVTTPosition = "50%"
	case 5: // middle-center
		sa.WebVTTPosition = "50%"
	case 6: // middle-right
		sa.WebVTTAlign = "right"
		sa.WebVTTPosition = "50%"
	case 1: // bottom-left
		sa.WebVTTAlign = "left"
		sa.WebVTTPosition = "90%"
	case 2: // bottom-center
		sa.WebVTTPosition = "90%"
	case 3: // bottom-right
		sa.WebVTTAlign = "right"
		sa.WebVTTPosition = "90%"
	}

	sa.WebVTTBold = sa.SRTBold
	sa.WebVTTItalics = sa.SRTItalics
	sa.WebVTTUnderline = sa.SRTUnderline

	sa.WebVTTTags = make([]WebVTTTag, 0)
	if sa.WebVTTBold {
		sa.WebVTTTags = append(sa.WebVTTTags, WebVTTTag{Name: "b"})
	}
	if sa.WebVTTItalics {
		sa.WebVTTTags = append(sa.WebVTTTags, WebVTTTag{Name: "i"})
	}
	if sa.WebVTTUnderline {
		sa.WebVTTTags = append(sa.WebVTTTags, WebVTTTag{Name: "u"})
	}
}

func (sa *StyleAttributes) propagateSSAAttributes() {}

func (sa *StyleAttributes) propagateSTLAttributes() {
	if sa.STLJustification != nil {
		switch *sa.STLJustification {
		case JustificationCentered:
			// default to middle anyway?
		case JustificationRight:
			sa.WebVTTAlign = "right"
		case JustificationLeft:
			sa.WebVTTAlign = "left"
		}
	}
	// converts STL vertical position (row number) to WebVTT line percentage
	if sa.STLPosition != nil && sa.STLPosition.MaxRows > 0 {
		// in-vision vertical position ranges from 0 to maxrows (maxrows <= 99)
		sa.WebVTTLine = fmt.Sprintf("%d%%", sa.STLPosition.VerticalPosition*100/sa.STLPosition.MaxRows)
		// teletext vertical position ranges from 1 to 23; as webvtt line percentage starts
		// from the top at 0%, substract 1 to the stl position to get a better conversion.
		// Especially apparent on Shaka player, where a single line at vp 22 would be half
		// out of bounds at 95% (22*100/23), and fine at 91% (21*100/23)
		if sa.STLPosition.MaxRows == 23 && sa.STLPosition.VerticalPosition > 0 {
			sa.WebVTTLine = fmt.Sprintf("%d%%", (sa.STLPosition.VerticalPosition-1)*100/sa.STLPosition.MaxRows)
		}
	}
}

func (sa *StyleAttributes) propagateTeletextAttributes() {
	if sa.TeletextColor != nil {
		sa.TTMLColor = astikit.StrPtr("#" + sa.TeletextColor.TTMLString())
	}
}

// reference for migration: https://w3c.github.io/ttml-webvtt-mapping/
func (sa *StyleAttributes) propagateTTMLAttributes() {
	if sa.TTMLTextAlign != nil {
		sa.WebVTTAlign = *sa.TTMLTextAlign
	}
	if sa.TTMLExtent != nil {
		//region settings
		lineHeight := 5 //assuming height of line as 5.33vh
		dimensions := strings.Split(*sa.TTMLExtent, " ")
		if len(dimensions) > 1 {
			sa.WebVTTWidth = dimensions[0]
			if height, err := strconv.Atoi(strings.ReplaceAll(dimensions[1], "%", "")); err == nil {
				sa.WebVTTLines = height / lineHeight
			}
			//cue settings
			//default TTML WritingMode is lrtb i.e. left to right, top to bottom
			sa.WebVTTSize = dimensions[1]
			if sa.TTMLWritingMode != nil && strings.HasPrefix(*sa.TTMLWritingMode, "tb") {
				sa.WebVTTSize = dimensions[0]
			}
		}
	}
	if sa.TTMLOrigin != nil {
		//region settings
		sa.WebVTTRegionAnchor = "0%,0%"
		sa.WebVTTViewportAnchor = strings.ReplaceAll(strings.TrimSpace(*sa.TTMLOrigin), " ", ",")
		sa.WebVTTScroll = "up"
		//cue settings
		coordinates := strings.Split(*sa.TTMLOrigin, " ")
		if len(coordinates) > 1 {
			sa.WebVTTLine = coordinates[0]
			sa.WebVTTPosition = coordinates[1]
			if sa.TTMLWritingMode != nil && strings.HasPrefix(*sa.TTMLWritingMode, "tb") {
				sa.WebVTTLine = coordinates[1]
				sa.WebVTTPosition = coordinates[0]
			}
		}
	}
}

func (sa *StyleAttributes) propagateWebVTTAttributes() {
	// copy relevant attrs to SRT ones
	if sa.TTMLColor != nil {
		sa.SRTColor = sa.TTMLColor
	}
	sa.SRTBold = sa.WebVTTBold
	sa.SRTItalics = sa.WebVTTItalics
	sa.SRTUnderline = sa.WebVTTUnderline
}

// Metadata represents metadata
// TODO Merge attributes
type Metadata struct {
	Comments                                            []string
	Framerate                                           int
	Language                                            string
	SSACollisions                                       string
	SSAOriginalEditing                                  string
	SSAOriginalScript                                   string
	SSAOriginalTiming                                   string
	SSAOriginalTranslation                              string
	SSAPlayDepth                                        *int
	SSAPlayResX, SSAPlayResY                            *int
	SSAScriptType                                       string
	SSAScriptUpdatedBy                                  string
	SSASynchPoint                                       string
	SSATimer                                            *float64
	SSAUpdateDetails                                    string
	SSAWrapStyle                                        string
	SSAScaledBorderAndShadow                            bool
	STLCountryOfOrigin                                  string
	STLCreationDate                                     *time.Time
	STLDisplayStandardCode                              string
	STLEditorContactDetails                             string
	STLEditorName                                       string
	STLMaximumNumberOfDisplayableCharactersInAnyTextRow *int
	STLMaximumNumberOfDisplayableRows                   *int
	STLOriginalEpisodeTitle                             string
	STLPublisher                                        string
	STLRevisionDate                                     *time.Time
	STLRevisionNumber                                   int
	STLSubtitleListReferenceCode                        string
	STLTimecodeStartOfProgramme                         time.Duration
	STLTranslatedEpisodeTitle                           string
	STLTranslatedProgramTitle                           string
	STLTranslatorContactDetails                         string
	STLTranslatorName                                   string
	Title                                               string
	TTMLCopyright                                       string
	WebVTTTimestampMap                                  *WebVTTTimestampMap
}

// Region represents a subtitle's region
type Region struct {
	ID          string
	InlineStyle *StyleAttributes
	Style       *Style
}

// Style represents a subtitle's style
type Style struct {
	ID          string
	InlineStyle *StyleAttributes
	Style       *Style
}

// Line represents a set of formatted line items
type Line struct {
	Items     []LineItem
	VoiceName string
}

// String implement the Stringer interface
func (l Line) String() string {
	var texts []string
	for _, i := range l.Items {
		texts = append(texts, i.Text)
	}
	// Don't add spaces here since items must contain their own space
	return strings.Join(texts, "")
}

// LineItem represents a formatted line item
type LineItem struct {
	InlineStyle *StyleAttributes
	StartAt     time.Duration
	Style       *Style
	Text        string
}

// Add adds a duration to each time boundaries. As in the time package, duration can be negative.
func (s *Subtitles) Add(d time.Duration) {
	for idx := 0; idx < len(s.Items); idx++ {
		s.Items[idx].EndAt += d
		s.Items[idx].StartAt += d
		if s.Items[idx].EndAt <= 0 && s.Items[idx].StartAt <= 0 {
			s.Items = append(s.Items[:idx], s.Items[idx+1:]...)
			idx--
		} else if s.Items[idx].StartAt <= 0 {
			s.Items[idx].StartAt = time.Duration(0)
		}
	}
}

// Duration returns the subtitles duration
func (s Subtitles) Duration() time.Duration {
	if len(s.Items) == 0 {
		return time.Duration(0)
	}
	return s.Items[len(s.Items)-1].EndAt
}

// ForceDuration updates the subtitles duration.
// If requested duration is bigger, then we create a dummy item.
// If requested duration is smaller, then we remove useless items and we cut the last item or add a dummy item.
func (s *Subtitles) ForceDuration(d time.Duration, addDummyItem bool) {
	// Requested duration is the same as the subtitles'one
	if s.Duration() == d {
		return
	}

	// Requested duration is bigger than subtitles'one
	if s.Duration() > d {
		// Find last item before input duration and update end at
		var lastIndex = -1
		for index, i := range s.Items {
			// Start at is bigger than input duration, we've found the last item
			if i.StartAt >= d {
				lastIndex = index
				break
			} else if i.EndAt > d {
				s.Items[index].EndAt = d
			}
		}

		// Last index has been found
		if lastIndex != -1 {
			s.Items = s.Items[:lastIndex]
		}
	}

	// Add dummy item with the minimum duration possible
	if addDummyItem && s.Duration() < d {
		s.Items = append(s.Items, &Item{EndAt: d, Lines: []Line{{Items: []LineItem{{Text: "..."}}}}, StartAt: d - time.Millisecond})
	}
}

// Fragment fragments subtitles with a specific fragment duration
func (s *Subtitles) Fragment(f time.Duration) {
	// Nothing to fragment
	if len(s.Items) == 0 {
		return
	}

	// Here we want to simulate fragments of duration f until there are no subtitles left in that period of time
	var fragmentStartAt, fragmentEndAt = time.Duration(0), f
	for fragmentStartAt < s.Items[len(s.Items)-1].EndAt {
		// We loop through subtitles and process the ones that either contain the fragment start at,
		// or contain the fragment end at
		//
		// It's useless processing subtitles contained between fragment start at and end at
		//             |____________________|             <- subtitle
		//           |                        |
		//   fragment start at        fragment end at
		for i, sub := range s.Items {
			// Init
			var newSub = &Item{}
			*newSub = *sub

			// A switch is more readable here
			switch {
			// Subtitle contains fragment start at
			// |____________________|                         <- subtitle
			//           |                        |
			//   fragment start at        fragment end at
			case sub.StartAt < fragmentStartAt && sub.EndAt > fragmentStartAt:
				sub.StartAt = fragmentStartAt
				newSub.EndAt = fragmentStartAt
			// Subtitle contains fragment end at
			//                         |____________________| <- subtitle
			//           |                        |
			//   fragment start at        fragment end at
			case sub.StartAt < fragmentEndAt && sub.EndAt > fragmentEndAt:
				sub.StartAt = fragmentEndAt
				newSub.EndAt = fragmentEndAt
			default:
				continue
			}

			// Insert new sub
			s.Items = append(s.Items[:i], append([]*Item{newSub}, s.Items[i:]...)...)
		}

		// Update fragments boundaries
		fragmentStartAt += f
		fragmentEndAt += f
	}

	// Order
	s.Order()
}

// IsEmpty returns whether the subtitles are empty
func (s Subtitles) IsEmpty() bool {
	return len(s.Items) == 0
}

// Merge merges subtitles i into subtitles
func (s *Subtitles) Merge(i *Subtitles) {
	// Append items
	s.Items = append(s.Items, i.Items...)
	s.Order()

	// Add regions
	for _, region := range i.Regions {
		if _, ok := s.Regions[region.ID]; !ok {
			s.Regions[region.ID] = region
		}
	}

	// Add styles
	for _, style := range i.Styles {
		if _, ok := s.Styles[style.ID]; !ok {
			s.Styles[style.ID] = style
		}
	}
}

// Optimize optimizes subtitles
func (s *Subtitles) Optimize() {
	// Nothing to optimize
	if len(s.Items) == 0 {
		return
	}

	// Remove unused regions and style
	s.removeUnusedRegionsAndStyles()
}

// removeUnusedRegionsAndStyles removes unused regions and styles
func (s *Subtitles) removeUnusedRegionsAndStyles() {
	// Loop through items
	var usedRegions, usedStyles = make(map[string]bool), make(map[string]bool)
	for _, item := range s.Items {
		// Add region
		if item.Region != nil {
			usedRegions[item.Region.ID] = true
		}

		// Add style
		if item.Style != nil {
			usedStyles[item.Style.ID] = true
		}

		// Loop through lines
		for _, line := range item.Lines {
			// Loop through line items
			for _, lineItem := range line.Items {
				// Add style
				if lineItem.Style != nil {
					usedStyles[lineItem.Style.ID] = true
				}
			}
		}
	}

	// Loop through regions
	for id, region := range s.Regions {
		if _, ok := usedRegions[region.ID]; ok {
			if region.Style != nil {
				usedStyles[region.Style.ID] = true
			}
		} else {
			delete(s.Regions, id)
		}
	}

	// Loop through style
	for id, style := range s.Styles {
		if _, ok := usedStyles[style.ID]; !ok {
			delete(s.Styles, id)
		}
	}
}

// Order orders items
func (s *Subtitles) Order() {
	// Nothing to do if less than 1 element
	if len(s.Items) <= 1 {
		return
	}

	// Order
	sort.SliceStable(s.Items, func(i, j int) bool {
		return s.Items[i].StartAt < s.Items[j].StartAt
	})
}

// RemoveStyling removes the styling from the subtitles
func (s *Subtitles) RemoveStyling() {
	s.Regions = map[string]*Region{}
	s.Styles = map[string]*Style{}
	for _, i := range s.Items {
		i.Region = nil
		i.Style = nil
		i.InlineStyle = nil
		for idxLine, l := range i.Lines {
			for idxLineItem := range l.Items {
				i.Lines[idxLine].Items[idxLineItem].InlineStyle = nil
				i.Lines[idxLine].Items[idxLineItem].Style = nil
			}
		}
	}
}

// Unfragment unfragments subtitles
func (s *Subtitles) Unfragment() {
	// Nothing to do if less than 1 element
	if len(s.Items) <= 1 {
		return
	}

	// Order
	s.Order()

	// Loop through items
	for i := 0; i < len(s.Items)-1; i++ {
		for j := i + 1; j < len(s.Items); j++ {
			// Items are the same
			if s.Items[i].String() == s.Items[j].String() && s.Items[i].EndAt >= s.Items[j].StartAt {
				// Only override end time if longer
				if s.Items[i].EndAt < s.Items[j].EndAt {
					s.Items[i].EndAt = s.Items[j].EndAt
				}
				s.Items = append(s.Items[:j], s.Items[j+1:]...)
				j--
			} else if s.Items[i].EndAt < s.Items[j].StartAt {
				break
			}
		}
	}
}

// ApplyLinearCorrection applies linear correction
func (s *Subtitles) ApplyLinearCorrection(actual1, desired1, actual2, desired2 time.Duration) {
	// Get parameters
	a := float64(desired2-desired1) / float64(actual2-actual1)
	b := time.Duration(float64(desired1) - a*float64(actual1))

	// Loop through items
	for idx := range s.Items {
		s.Items[idx].EndAt = time.Duration(a*float64(s.Items[idx].EndAt)) + b
		s.Items[idx].StartAt = time.Duration(a*float64(s.Items[idx].StartAt)) + b
	}
}

// Write writes subtitles to a file
func (s Subtitles) Write(dst string) (err error) {
	// Create the file
	var f *os.File
	if f, err = os.Create(dst); err != nil {
		err = fmt.Errorf("astisub: creating %s failed: %w", dst, err)
		return
	}
	defer f.Close()

	// Write the content
	switch filepath.Ext(strings.ToLower(dst)) {
	case ".srt":
		err = s.WriteToSRT(f)
	case ".ssa", ".ass":
		err = s.WriteToSSA(f)
	case ".stl":
		err = s.WriteToSTL(f)
	case ".ttml":
		err = s.WriteToTTML(f)
	case ".vtt":
		err = s.WriteToWebVTT(f)
	default:
		err = ErrInvalidExtension
	}
	return
}

// parseDuration parses a duration in "00:00:00.000", "00:00:00,000" or "0:00:00:00" format
func parseDuration(i, millisecondSep string, numberOfMillisecondDigits int) (o time.Duration, err error) {
	// Split milliseconds
	var parts = strings.Split(i, millisecondSep)
	var milliseconds int
	var s string
	if len(parts) >= 2 {
		// Invalid number of millisecond digits
		s = strings.TrimSpace(parts[len(parts)-1])
		if len(s) > 3 {
			err = fmt.Errorf("astisub: Invalid number of millisecond digits detected in %s", i)
			return
		}

		// Parse milliseconds
		if milliseconds, err = strconv.Atoi(s); err != nil {
			err = fmt.Errorf("astisub: atoi of %s failed: %w", s, err)
			return
		}
		milliseconds *= int(math.Pow10(numberOfMillisecondDigits - len(s)))
		s = strings.Join(parts[:len(parts)-1], millisecondSep)
	} else {
		s = i
	}

	// Split hours, minutes and seconds
	parts = strings.Split(strings.TrimSpace(s), ":")
	var partSeconds, partMinutes, partHours string
	if len(parts) == 2 {
		partSeconds = parts[1]
		partMinutes = parts[0]
	} else if len(parts) == 3 {
		partSeconds = parts[2]
		partMinutes = parts[1]
		partHours = parts[0]
	} else {
		err = fmt.Errorf("astisub: No hours, minutes or seconds detected in %s", i)
		return
	}

	// Parse seconds
	var seconds int
	s = strings.TrimSpace(partSeconds)
	if seconds, err = strconv.Atoi(s); err != nil {
		err = fmt.Errorf("astisub: atoi of %s failed: %w", s, err)
		return
	}

	// Parse minutes
	var minutes int
	s = strings.TrimSpace(partMinutes)
	if minutes, err = strconv.Atoi(s); err != nil {
		err = fmt.Errorf("astisub: atoi of %s failed: %w", s, err)
		return
	}

	// Parse hours
	var hours int
	if len(partHours) > 0 {
		s = strings.TrimSpace(partHours)
		if hours, err = strconv.Atoi(s); err != nil {
			err = fmt.Errorf("astisub: atoi of %s failed: %w", s, err)
			return
		}
	}

	// Generate output
	o = time.Duration(milliseconds)*time.Millisecond + time.Duration(seconds)*time.Second + time.Duration(minutes)*time.Minute + time.Duration(hours)*time.Hour
	return
}

// formatDuration formats a duration
func formatDuration(i time.Duration, millisecondSep string, numberOfMillisecondDigits int) (s string) {
	// Parse hours
	var hours = int(i / time.Hour)
	var n = i % time.Hour
	if hours < 10 {
		s += "0"
	}
	s += strconv.Itoa(hours) + ":"

	// Parse minutes
	var minutes = int(n / time.Minute)
	n = i % time.Minute
	if minutes < 10 {
		s += "0"
	}
	s += strconv.Itoa(minutes) + ":"

	// Parse seconds
	var seconds = int(n / time.Second)
	n = i % time.Second
	if seconds < 10 {
		s += "0"
	}
	s += strconv.Itoa(seconds) + millisecondSep

	// Parse milliseconds
	var milliseconds = math.Floor(float64(n) / float64(time.Millisecond) / float64(math.Pow(10, 3-float64(numberOfMillisecondDigits))))
	s += astikit.StrPad(strconv.FormatFloat(milliseconds, 'f', 0, 64), '0', numberOfMillisecondDigits, astikit.PadLeft)
	return
}

// appendStringToBytesWithNewLine adds a string to bytes then adds a new line
func appendStringToBytesWithNewLine(i []byte, s string) (o []byte) {
	o = append(i, []byte(s)...)
	o = append(o, bytesLineSeparator...)
	return
}

func htmlTokenAttribute(t *html.Token, key string) *string {

	for _, attr := range t.Attr {
		if attr.Key == key {
			return &attr.Val
		}
	}

	return nil
}

func escapeHTML(i string) string {
	return htmlEscaper.Replace(i)
}

func unescapeHTML(i string) string {
	return htmlUnescaper.Replace(i)
}

func newScanner(i io.Reader) *bufio.Scanner {
	var scanner = bufio.NewScanner(i)
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) == 0 {
			return 0, nil, nil
		}
		if i := bytes.IndexAny(data, "\r\n"); i >= 0 {
			if data[i] == '\n' {
				// We have a line terminated by single newline.
				return i + 1, data[0:i], nil
			}
			advance = i + 1
			if len(data) > i+1 && data[i+1] == '\n' {
				advance += 1
			}
			return advance, data[0:i], nil
		}
		// If we're at EOF, we have a final, non-terminated line. Return it.
		if atEOF {
			return len(data), data, nil
		}
		// Request more data.
		return 0, nil, nil
	})
	return scanner
}
