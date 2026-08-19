package renderers

import (
	"bytes"
	"regexp"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var kramdownIALCandidateRE = regexp.MustCompile(`\{:[^{}\n]*\}`)

type kramdownIALTransformer struct{}

func (kramdownIALTransformer) Transform(document *ast.Document, reader text.Reader, _ parser.Context) {
	source := reader.Source()
	transformBlockIALs(document, source)
	transformInlineIALs(document, source)
}

func transformBlockIALs(document *ast.Document, source []byte) {
	var blocks []ast.Node
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch node.(type) {
			case *ast.Paragraph, *ast.TextBlock:
				blocks = append(blocks, node)
			}
		}
		return ast.WalkContinue, nil
	})

	for _, block := range blocks {
		if block.Parent() == nil || block.Lines().Len() == 0 {
			continue
		}
		transformBlockIALLines(block, source)
	}
}

func transformBlockIALLines(block ast.Node, source []byte) {
	lines := block.Lines()
	leading := parsedIALLines(lines, source, true)
	trailing := parsedIALLines(lines, source, false)

	if leading == 0 && trailing == 0 {
		return
	}
	if leading == lines.Len() {
		attrs := attributesFromLines(lines, 0, lines.Len(), source)
		target := standaloneIALTarget(block, lines.At(0).Start, source)
		if target == nil {
			return
		}
		applyIALAttributes(target, attrs)
		block.Parent().RemoveChild(block.Parent(), block)
		return
	}

	if leading > 0 {
		start := lines.At(0).Start
		attrs := attributesFromLines(lines, 0, leading, source)
		end := lines.At(leading - 1).Stop
		applyIALAttributes(containingListOr(block, start, source), attrs)
		removeBlockSourceRange(block, start, end)
		lines.SetSliced(leading, lines.Len())
	}

	// Recompute the trailing count after slicing leading marker lines.
	trailing = parsedIALLines(lines, source, false)
	if trailing == 0 || trailing == lines.Len() {
		return
	}
	start := lines.At(lines.Len() - trailing).Start
	end := lines.At(lines.Len() - 1).Stop
	attrs := attributesFromLines(lines, lines.Len()-trailing, lines.Len(), source)
	applyIALAttributes(containingListOr(block, start, source), attrs)
	removeBlockSourceRange(block, start, end)
	lines.SetSliced(0, lines.Len()-trailing)
}

func parsedIALLines(lines *text.Segments, source []byte, fromStart bool) int {
	count := 0
	for count < lines.Len() {
		index := count
		if !fromStart {
			index = lines.Len() - 1 - count
		}
		line := lines.At(index)
		if _, ok := parseKramdownIAL(line.Value(source)); !ok {
			break
		}
		count++
	}
	return count
}

func attributesFromLines(lines *text.Segments, start, stop int, source []byte) parser.Attributes {
	var result parser.Attributes
	for i := start; i < stop; i++ {
		line := lines.At(i)
		attrs, _ := parseKramdownIAL(line.Value(source))
		result = append(result, attrs...)
	}
	return result
}

func standaloneIALTarget(block ast.Node, markerStart int, source []byte) ast.Node {
	previous := block.PreviousSibling()
	next := block.NextSibling()
	if previousPhysicalLineHasContent(source, markerStart) {
		if previous != nil {
			return previous
		}
		return next
	}
	if next != nil {
		return next
	}
	return previous
}

func containingListOr(block ast.Node, markerStart int, source []byte) ast.Node {
	item, ok := block.Parent().(*ast.ListItem)
	if !ok || item.NextSibling() != nil || sourceLineIndent(source, markerStart) != 0 {
		return block
	}
	if list, ok := item.Parent().(*ast.List); ok {
		return list
	}
	return block
}

func sourceLineIndent(source []byte, offset int) int {
	lineStart := bytes.LastIndexByte(source[:offset], '\n') + 1
	return offset - lineStart
}

func previousPhysicalLineHasContent(source []byte, offset int) bool {
	if offset <= 0 {
		return false
	}
	before := source[:offset]
	before = bytes.TrimSuffix(before, []byte("\n"))
	before = bytes.TrimSuffix(before, []byte("\r"))
	if index := bytes.LastIndexByte(before, '\n'); index >= 0 {
		before = before[index+1:]
	}
	return len(bytes.TrimSpace(before)) > 0
}

func removeBlockSourceRange(block ast.Node, start, stop int) {
	for child := block.FirstChild(); child != nil; {
		next := child.NextSibling()
		segment, ok := child.(*ast.Text)
		if ok && segment.Segment.Start < stop && segment.Segment.Stop > start {
			switch {
			case segment.Segment.Start >= start && segment.Segment.Stop <= stop:
				block.RemoveChild(block, child)
			case segment.Segment.Start < start:
				segment.Segment = segment.Segment.WithStop(start)
				segment.SetSoftLineBreak(false)
				segment.SetHardLineBreak(false)
			case segment.Segment.Stop > stop:
				segment.Segment = segment.Segment.WithStart(stop)
			}
		}
		child = next
	}

	if last, ok := block.LastChild().(*ast.Text); ok {
		last.SetSoftLineBreak(false)
		last.SetHardLineBreak(false)
	}
}

func transformInlineIALs(document *ast.Document, source []byte) {
	_ = ast.Walk(document, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || !isInlineIALContainer(node) {
			return ast.WalkContinue, nil
		}
		var children []ast.Node
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			children = append(children, child)
		}
		for _, child := range children {
			if child.Parent() == node {
				transformInlineIAL(node, child, source)
			}
		}
		return ast.WalkSkipChildren, nil
	})
}

func isInlineIALContainer(node ast.Node) bool {
	switch node.(type) {
	case *ast.Paragraph, *ast.Heading, *ast.TextBlock:
		return true
	default:
		return false
	}
}

func transformInlineIAL(parent, node ast.Node, source []byte) {
	textNode, ok := node.(*ast.Text)
	target := node.PreviousSibling()
	if !ok || target == nil || parent.Lines().Len() == 0 {
		return
	}

	stop := parent.Lines().At(parent.Lines().Len() - 1).Stop
	value := source[textNode.Segment.Start:stop]
	match := kramdownIALCandidateRE.FindIndex(value)
	if match == nil || len(bytes.TrimSpace(value[:match[0]])) != 0 {
		return
	}
	attrs, ok := parseKramdownIAL(value[match[0]:match[1]])
	if !ok {
		return
	}

	applyIALAttributes(target, attrs)
	consumed := match[1]
	for consumed < len(value) && (value[consumed] == ' ' || value[consumed] == '\t') {
		consumed++
	}
	removeInlineSourceRange(parent, textNode.Segment.Start, textNode.Segment.Start+consumed)
}

func removeInlineSourceRange(parent ast.Node, start, stop int) {
	for child := parent.FirstChild(); child != nil; {
		next := child.NextSibling()
		textNode, ok := child.(*ast.Text)
		if ok && textNode.Segment.Start < stop && textNode.Segment.Stop > start {
			switch {
			case textNode.Segment.Start >= start && textNode.Segment.Stop <= stop:
				parent.RemoveChild(parent, child)
			case textNode.Segment.Start < start:
				textNode.Segment = textNode.Segment.WithStop(start)
			case textNode.Segment.Stop > stop:
				textNode.Segment = textNode.Segment.WithStart(stop)
			}
		}
		child = next
	}
}

func parseKramdownIAL(value []byte) (parser.Attributes, bool) {
	value = bytes.TrimSpace(value)
	if len(value) < 4 || !bytes.HasPrefix(value, []byte("{:")) || value[len(value)-1] != '}' {
		return nil, false
	}

	pandoc := make([]byte, 0, len(value)-1)
	pandoc = append(pandoc, '{')
	pandoc = append(pandoc, value[2:]...)
	reader := text.NewReader(pandoc)
	attrs, ok := parser.ParseAttributes(reader)
	if !ok || len(attrs) == 0 {
		return nil, false
	}
	remaining, _ := reader.PeekLine()
	if len(bytes.TrimSpace(remaining)) != 0 {
		return nil, false
	}
	return attrs, true
}

func applyIALAttributes(node ast.Node, attrs parser.Attributes) {
	for _, attr := range attrs {
		value := attr.Value
		if bytes.Equal(attr.Name, []byte("class")) {
			value = mergedClassAttribute(node, attr.Value)
		}
		node.SetAttribute(attr.Name, value)
	}
}

func mergedClassAttribute(node ast.Node, added interface{}) interface{} {
	addedBytes, ok := added.([]byte)
	if !ok {
		return added
	}
	existing, ok := node.AttributeString("class")
	if !ok {
		return addedBytes
	}
	existingBytes, ok := existing.([]byte)
	if !ok || len(existingBytes) == 0 {
		return addedBytes
	}
	merged := make([]byte, 0, len(existingBytes)+1+len(addedBytes))
	merged = append(merged, existingBytes...)
	merged = append(merged, ' ')
	merged = append(merged, addedBytes...)
	return merged
}
