// Copyright 2021 Jon Engelsman
// modified 24/14/2021 by Jon Engelsman
// Copyright 2017 The oksvg Authors. All rights reserved.
// created: 2/12/2017 by S.R.Wiley

package svgg

import (
	"errors"
	"log"
	"unicode"

	"github.com/fogleman/gg"
)

// svgg is a modified version of srwiley/oksvg.
// It parses an svg path and draws the svg to a context
// using the fogleman/gg 2d rendering engine.

// original version of this file can be found here:
// https://github.com/srwiley/oksvg/blob/875f767ac39a9363479ee5c926bfbea2be68128c/svgp.go

//ErrorMode sets how the parser reacts to unparsed elements
type ErrorMode uint8

var (
	errParamMismatch  = errors.New("Param mismatch")
	errCommandUnknown = errors.New("Unknown command")
	errZeroLengthID   = errors.New("zero length id")
	errMissingID      = errors.New("cannot find id")
	errNotImplemented = errors.New("not implemented")
)

const (
	//IgnoreErrorMode skips unparsed SVG elements
	IgnoreErrorMode ErrorMode = iota
	//WarnErrorMode outputs a warning when an unparsed SVG element is found
	WarnErrorMode
	//StrictErrorMode causes a error when an unparsed SVG element is found
	StrictErrorMode
)

func reflect(px, py, rx, ry float64) (x, y float64) {
	return px*2 - rx, py*2 - ry
}

//Parser is used to parse SVG strings into drawing commands
type Parser struct {
	placeX, placeY         float64
	curX, curY             float64
	cntlPtX, cntlPtY       float64
	pathStartX, pathStartY float64
	points                 []float64
	lastKey                uint8
	ErrorMode              ErrorMode
	inPath                 bool
	dc                     *gg.Context
}

func NewParser(dc *gg.Context) *Parser {
	return &Parser{
		dc: dc,
	}
}

func (p *Parser) valsToAbs(last float64) {
	for i := 0; i < len(p.points); i++ {
		last += p.points[i]
		p.points[i] = last
	}
}

func (p *Parser) pointsToAbs(sz int) {
	lastX := p.placeX
	lastY := p.placeY
	for j := 0; j < len(p.points); j += sz {
		for i := 0; i < sz; i += 2 {
			p.points[i+j] += lastX
			p.points[i+1+j] += lastY
		}
		lastX = p.points[(j+sz)-2]
		lastY = p.points[(j+sz)-1]
	}
}

func (p *Parser) hasSetsOrMore(sz int, rel bool) bool {
	if !(len(p.points) >= sz && len(p.points)%sz == 0) {
		return false
	}
	if rel {
		p.pointsToAbs(sz)
	}
	return true
}

// ReadFloat reads a floating point value and adds it to the cursor's points slice.
func (p *Parser) ReadFloat(numStr string) error {
	last := 0
	isFirst := true
	for i, n := range numStr {
		if n == '.' {
			if isFirst == true {
				isFirst = false
				continue
			}
			f, err := parseFloat(numStr[last:i], 64)
			if err != nil {
				return err
			}
			p.points = append(p.points, f)
			last = i
		}
	}
	f, err := parseFloat(numStr[last:], 64)
	if err != nil {
		return err
	}
	p.points = append(p.points, f)
	return nil
}

// GetPoints reads a set of floating point values from the SVG format number string,
// and add them to the cursor's points slice.
func (p *Parser) GetPoints(dataPoints string) error {
	lastIndex := -1
	p.points = p.points[0:0]
	lr := ' '
	for i, r := range dataPoints {
		if unicode.IsNumber(r) == false && r != '.' && !(r == '-' && lr == 'e') && r != 'e' {
			if lastIndex != -1 {
				if err := p.ReadFloat(dataPoints[lastIndex:i]); err != nil {
					return err
				}
			}
			if r == '-' {
				lastIndex = i
			} else {
				lastIndex = -1
			}
		} else if lastIndex == -1 {
			lastIndex = i
		}
		lr = r
	}
	if lastIndex != -1 && lastIndex != len(dataPoints) {
		if err := p.ReadFloat(dataPoints[lastIndex:len(dataPoints)]); err != nil {
			return err
		}
	}
	return nil
}

func (p *Parser) reflectControlQuad() {
	switch p.lastKey {
	case 'q', 'Q', 'T', 't':
		p.cntlPtX, p.cntlPtY = reflect(p.placeX, p.placeY, p.cntlPtX, p.cntlPtY)
	default:
		p.cntlPtX, p.cntlPtY = p.placeX, p.placeY
	}
}

func (p *Parser) reflectControlCube() {
	switch p.lastKey {
	case 'c', 'C', 's', 'S':
		p.cntlPtX, p.cntlPtY = reflect(p.placeX, p.placeY, p.cntlPtX, p.cntlPtY)
	default:
		p.cntlPtX, p.cntlPtY = p.placeX, p.placeY
	}
}

// addSeg decodes an SVG seqment string and draws to the context.
func (p *Parser) addSeg(segString string) error {

	// Parse the string describing the numeric points in SVG format
	if err := p.GetPoints(segString[1:]); err != nil {
		return err
	}
	l := len(p.points)
	k := segString[0]
	rel := false
	switch k {
	case 'z':
		fallthrough
	case 'Z':
		if len(p.points) != 0 {
			return errParamMismatch
		}
		if p.inPath {
			// p.Path.Stop(true)
			p.placeX = p.pathStartX
			p.placeY = p.pathStartY
			p.inPath = false
		}
	case 'm':
		rel = true
		fallthrough
	case 'M':
		if !p.hasSetsOrMore(2, rel) {
			return errParamMismatch
		}
		p.pathStartX, p.pathStartY = p.points[0], p.points[1]
		p.inPath = true
		// p.Path.Start(fixed.Point26_6{X: fixed.Int26_6((p.pathStartX + p.curX) * 64), Y: fixed.Int26_6((p.pathStartY + p.curY) * 64)})
		p.dc.MoveTo(p.points[0], p.points[1])

		for i := 2; i < l-1; i += 2 {
			p.dc.LineTo(p.points[i], p.points[i+1])
			// p.Path.Line(fixed.Point26_6{
			// 	X: fixed.Int26_6((p.points[i] + p.curX) * 64),
			// 	Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)})
		}
		p.placeX = p.points[l-2]
		p.placeY = p.points[l-1]
	case 'l':
		rel = true
		fallthrough
	case 'L':
		return errNotImplemented

		// if !p.hasSetsOrMore(2, rel) {
		// 	return errParamMismatch
		// }
		// for i := 0; i < l-1; i += 2 {
		// 	// p.Path.Line(fixed.Point26_6{
		// 	// 	X: fixed.Int26_6((p.points[i] + p.curX) * 64),
		// 	// 	Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)})
		// }
		// p.placeX = p.points[l-2]
		// p.placeY = p.points[l-1]
	case 'v':
		p.valsToAbs(p.placeY)
		fallthrough
	case 'V':
		return errNotImplemented

		// if !p.hasSetsOrMore(1, false) {
		// 	return errParamMismatch
		// }
		// for _, p := range p.points {
		// 	_ = p
		// 	// p.Path.Line(fixed.Point26_6{
		// 	// 	X: fixed.Int26_6((p.placeX + p.curX) * 64),
		// 	// 	Y: fixed.Int26_6((p + p.curY) * 64)})
		// }
		// p.placeY = p.points[l-1]
	case 'h':
		p.valsToAbs(p.placeX)
		fallthrough
	case 'H':
		return errNotImplemented

		// if !p.hasSetsOrMore(1, false) {
		// 	return errParamMismatch
		// }
		// for _, p := range p.points {
		// 	_ = p
		// 	// p.Path.Line(fixed.Point26_6{
		// 	// 	X: fixed.Int26_6((p + p.curX) * 64),
		// 	// 	Y: fixed.Int26_6((p.placeY + p.curY) * 64)})
		// }
		// p.placeX = p.points[l-1]
	case 'q':
		rel = true
		fallthrough
	case 'Q':
		return errNotImplemented

		// if !p.hasSetsOrMore(4, rel) {
		// 	return errParamMismatch
		// }
		// for i := 0; i < l-3; i += 4 {
		// 	// p.Path.QuadBezier(
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i] + p.curX) * 64),
		// 	// 		Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)},
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i+2] + p.curX) * 64),
		// 	// 		Y: fixed.Int26_6((p.points[i+3] + p.curY) * 64)})
		// }
		// p.cntlPtX, p.cntlPtY = p.points[l-4], p.points[l-3]
		// p.placeX = p.points[l-2]
		// p.placeY = p.points[l-1]
	case 't':
		rel = true
		fallthrough
	case 'T':
		return errNotImplemented

		// // if !p.hasSetsOrMore(2, rel) {
		// // 	return errParamMismatch
		// // }
		// // for i := 0; i < l-1; i += 2 {
		// // 	p.reflectControlQuad()
		// // 	// p.Path.QuadBezier(
		// // 	// 	fixed.Point26_6{
		// // 	// 		X: fixed.Int26_6((p.cntlPtX + p.curX) * 64),
		// // 	// 		Y: fixed.Int26_6((p.cntlPtY + p.curY) * 64)},
		// // 	// 	fixed.Point26_6{
		// // 	// 		X: fixed.Int26_6((p.points[i] + p.curX) * 64),
		// // 	// 		Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)})
		// // 	p.lastKey = k
		// // 	p.placeX = p.points[i]
		// // 	p.placeY = p.points[i+1]
		// }
	case 'c':
		rel = true
		fallthrough
	case 'C':
		return errNotImplemented

		// if !p.hasSetsOrMore(6, rel) {
		// 	return errParamMismatch
		// }
		// for i := 0; i < l-5; i += 6 {
		// 	// p.Path.CubeBezier(
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i] + p.curX) * 64),
		// 	// 		Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)},
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i+2] + p.curX) * 64),
		// 	// 		Y: fixed.Int26_6((p.points[i+3] + p.curY) * 64)},
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i+4] + p.curX) * 64),
		// 	// 		Y: fixed.Int26_6((p.points[i+5] + p.curY) * 64)})
		// }
		// p.cntlPtX, p.cntlPtY = p.points[l-4], p.points[l-3]
		// p.placeX = p.points[l-2]
		// p.placeY = p.points[l-1]
	case 's':
		rel = true
		fallthrough
	case 'S':
		return errNotImplemented

		// if !p.hasSetsOrMore(4, rel) {
		// 	return errParamMismatch
		// }
		// for i := 0; i < l-3; i += 4 {
		// 	p.reflectControlCube()
		// 	// p.Path.CubeBezier(fixed.Point26_6{
		// 	// 	X: fixed.Int26_6((p.cntlPtX + p.curX) * 64), Y: fixed.Int26_6((p.cntlPtY + p.curY) * 64)},
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i] + p.curX) * 64), Y: fixed.Int26_6((p.points[i+1] + p.curY) * 64)},
		// 	// 	fixed.Point26_6{
		// 	// 		X: fixed.Int26_6((p.points[i+2] + p.curX) * 64), Y: fixed.Int26_6((p.points[i+3] + p.curY) * 64)})
		// 	p.lastKey = k
		// 	p.cntlPtX, p.cntlPtY = p.points[i], p.points[i+1]
		// 	p.placeX = p.points[i+2]
		// 	p.placeY = p.points[i+3]
		// }
	case 'a', 'A':
		return errNotImplemented

		// if !p.hasSetsOrMore(7, false) {
		// 	return errParamMismatch
		// }
		// for i := 0; i < l-6; i += 7 {
		// 	if k == 'a' {
		// 		p.points[i+5] += p.placeX
		// 		p.points[i+6] += p.placeY
		// 	}
		// 	p.AddArcFromA(p.points[i:])
		// }
	default:
		if p.ErrorMode == StrictErrorMode {
			return errCommandUnknown
		}
		if p.ErrorMode == WarnErrorMode {
			log.Println("Ignoring svg command " + string(k))
		}
	}
	// So we know how to extend some segment types
	p.lastKey = k
	return nil
}

//EllipseAt adds a path of an elipse centered at cx, cy of radius rx and ry
// to the Parser
func (p *Parser) EllipseAt(cx, cy, rx, ry float64) {
	log.Printf("warning: %s : %s\n", "EllipseAt", errNotImplemented.Error())
}

//AddArcFromA adds a path of an arc element to the Parser
func (p *Parser) AddArcFromA(points []float64) {
	log.Printf("warning: %s : %s\n", "AddArcFromA", errNotImplemented.Error())
}

func (p *Parser) init() {
	p.placeX = 0.0
	p.placeY = 0.0
	p.points = p.points[0:0]
	p.lastKey = ' '
	// p.Path.Clear()
	p.inPath = false
}

// CompilePath translates the svgPath description string and draws to the context.
// All valid SVG path elements are interpreted to fogleman/gg drawing commands.
func (p *Parser) CompilePath(svgPath string) error {
	p.init()
	lastIndex := -1
	for i, v := range svgPath {
		if unicode.IsLetter(v) && v != 'e' {
			if lastIndex != -1 {
				if err := p.addSeg(svgPath[lastIndex:i]); err != nil {
					return err
				}
			}
			lastIndex = i
		}
	}
	if lastIndex != -1 {
		if err := p.addSeg(svgPath[lastIndex:]); err != nil {
			return err
		}
	}

	p.dc.ClosePath()

	return nil
}

////////////////////////////////////////////////////////////
