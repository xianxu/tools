package main

import (
	"encoding/xml"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/xianxu/tools/cmd/define/store"
)

// Nodes retain ordered children, including text leaves, so unknown wrappers
// cannot hide or reorder source content. Leaf offsets address source.text.
type bilingualNode struct {
	class      string
	tag        string
	lang       store.Lang
	start, end int
	children   []*bilingualNode
}
type bilingualDocument struct {
	root   *bilingualNode
	source languageText
	native languageText
	title  string
}

// parseBilingualDocument shares a single class walk with provenance extraction.
// Identity validation bounds depth and rejects malformed or duplicate entries;
// encoding/xml never fetches the source's external DTD.
func parseBilingualDocument(record bilingualRecord) (bilingualDocument, error) {
	var doc bilingualDocument
	if len(record.HTML) > bilingualMaxBytes || len(record.Text) > bilingualMaxBytes {
		return doc, ErrBilingualLimit
	}
	id, err := bilingualRecordIdentity(record.HTML)
	if err != nil || !id.spanish || !utf8.ValidString(record.Text) || strings.TrimSpace(record.Text) == "" {
		return doc, ErrBilingualMalformed
	}
	doc.title = id.title
	decoder := xml.NewDecoder(strings.NewReader(record.HTML))
	var stack []*bilingualNode
	var source strings.Builder
	for {
		token, err := decoder.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return bilingualDocument{}, ErrBilingualMalformed
		}
		switch token := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 && !(token.Name.Local == "entry" && token.Name.Space == bilingualEntryNamespace) {
				continue
			}
			node := &bilingualNode{tag: token.Name.Local}
			if len(stack) > 0 {
				node.lang = stack[len(stack)-1].lang
				stack[len(stack)-1].children = append(stack[len(stack)-1].children, node)
			} else {
				doc.root = node
			}
			for _, attr := range token.Attr {
				if attr.Name.Local == "class" {
					node.class = attr.Value
				}
			}
			for _, class := range strings.Fields(node.class) {
				switch class {
				case "hw", "ex", "idm", "ind":
					node.lang = "es"
				case "trans":
					node.lang = "en"
				case "gp", "ph", "prx", "lg", "reg", "lev", "fld", "tgr", "ps", "sn", "underline":
					node.lang = ""
				}
			}
			stack = append(stack, node)
		case xml.EndElement:
			if len(stack) > 0 {
				stack = stack[:len(stack)-1]
			}
		case xml.CharData:
			if len(stack) == 0 {
				continue
			}
			parent := stack[len(stack)-1]
			leaf := &bilingualNode{lang: parent.lang, start: source.Len()}
			source.Write(token)
			leaf.end = source.Len()
			parent.children = append(parent.children, leaf)
			if leaf.lang != "" {
				doc.source.spans = append(doc.source.spans, languageSpan{start: leaf.start, end: leaf.end, lang: leaf.lang})
			}
		}
	}
	doc.source.text = source.String()
	// Only whitespace normalization is allowed; removing an interior word boundary
	// or matching a later repeated spelling cannot establish correspondence.
	if strings.Join(strings.Fields(doc.source.text), " ") != strings.Join(strings.Fields(record.Text), " ") {
		return bilingualDocument{}, ErrBilingualMalformed
	}
	doc.native = projectDictionaryText(doc.source, record.Text)
	return doc, nil
}

func (n *bilingualNode) has(class string) bool {
	for _, c := range strings.Fields(n.class) {
		if c == class {
			return true
		}
	}
	return false
}

// renderBilingualDocument inserts breaks only at source structure boundaries.
// trg remains inline inside exg: examples and translations form one logical row.
// Presentation tint belongs to the enclosing dictionary section, not these runs.
func renderBilingualDocument(doc bilingualDocument, opt RenderOpts) (string, []Region) {
	p := newPalette(opt.Color)
	var output strings.Builder
	type run struct {
		text, style string
		prose       bool
	}
	var row []run
	rowIndent := 0
	flush := func() {
		// Trim source whitespace before inserting escapes, so color cannot alter
		// row boundaries or leave spaces hidden behind a final style reset.
		for len(row) > 0 {
			row[0].text = strings.TrimLeftFunc(row[0].text, unicode.IsSpace)
			if row[0].text != "" {
				break
			}
			row = row[1:]
		}
		for len(row) > 0 {
			i := len(row) - 1
			row[i].text = strings.TrimRightFunc(row[i].text, unicode.IsSpace)
			if row[i].text != "" {
				break
			}
			row = row[:i]
		}
		if len(row) > 0 {
			var styled strings.Builder
			for _, r := range row {
				text := r.text
				if r.prose {
					text = opt.prose(text, r.style)
				}
				if r.style != "" {
					styled.WriteString(r.style)
				}
				styled.WriteString(text)
				if r.style != "" {
					styled.WriteString(p.off)
				}
			}
			output.WriteString(strings.Repeat(" ", rowIndent))
			output.WriteString(wrapText(styled.String(), opt.Width, rowIndent))
			output.WriteByte('\n')
		}
		row = nil
	}

	var walk func(*bilingualNode, int, string, bool)
	walk = func(n *bilingualNode, depth int, style string, prose bool) {
		if n.tag == "" {
			row = append(row, run{doc.source.text[n.start:n.end], style, prose})
			return
		}
		block := n.has("gramb") || n.has("semb") || n.has("exg") || n.has("idmb") || n.has("idmsec")
		if block {
			flush()
			depth++
			rowIndent = depth * 2
		}
		switch {
		case n.has("hw"):
			style = p.head
			prose = false
		case n.has("ps"):
			style = p.pos
			prose = false
		case n.has("sn"):
			style = p.num
			prose = false
		case n.has("ex"):
			style = p.ex
			prose = true
		case n.has("idm"):
			style = p.sect
			prose = true
		case n.has("trans"), n.has("ind"), n.has("co"):
			style = ""
			prose = true
		case n.has("lg"), n.has("reg"), n.has("gp"):
			style = p.dim
			prose = false
		}
		if opt.Color {
			if n.has("bold") {
				style += "\x1b[1m"
			}
			if n.has("italic") {
				style += "\x1b[3m"
			}
			if n.has("underline") {
				style += "\x1b[4m"
			}
			switch n.tag {
			case "b", "strong":
				style += "\x1b[1m"
			case "i", "em":
				style += "\x1b[3m"
			case "u":
				style += "\x1b[4m"
			}
		}
		for _, child := range n.children {
			walk(child, depth, style, prose)
		}
		if block {
			flush()
			rowIndent = max(0, depth-1) * 2
		}
	}
	if doc.root == nil {
		return "", nil
	}
	walk(doc.root, 0, "", false)
	flush()
	out := output.String()
	entry := Entry{Head: []HeadTok{{Kind: HeadWord, Text: doc.title}}}
	return out, regionsIn(entry, out, opt.Word)
}
