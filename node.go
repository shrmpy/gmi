package gmi

import (
	"fmt"
	"net/url"
	"strings"
)

var textFormat = "%s" // Changed to "%q" in tests for better error messages.

// nodes are the tree elements created by parse logic
type Node interface {
	Type() NodeType
	String() string
	writeTo(*strings.Builder)
}

type NodeType int

// Pos represents a byte position in the original input text from which
// this text was parsed.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// Type returns itself and provides an easy default implementation
// for embedding in a Node. Embedded in all non-trivial Nodes.
func (t NodeType) Type() NodeType {
	return t
}

const (
	NodeText    NodeType = iota // Plain text.
	NodeLink                    // Link.
	NodeList                    // A list of nodes.
	gmPretoggle                 // Preformat toggle switch.
	gmPreformat                 // Preformat text body.
	gmBlank
)

// Nodes.

// ListNode represents a sub/tree as a sequence of nodes.
type ListNode struct {
	NodeType
	Pos
	Nodes []Node // The element nodes in lexical order.
}

func (t *Tree) newList(pos Pos) *ListNode {
	return &ListNode{NodeType: NodeList, Pos: pos}
}
func (l *ListNode) append(n Node) {
	l.Nodes = append(l.Nodes, n)
}
func (l *ListNode) String() string {
	var sb strings.Builder
	l.writeTo(&sb)
	return sb.String()
}
func (l *ListNode) writeTo(sb *strings.Builder) {
	for _, n := range l.Nodes {
		n.writeTo(sb)
	}
}

// TextNode holds plain text.
type TextNode struct {
	NodeType
	Pos
	Text []byte
}

func (t *Tree) newText(pos Pos, text string) *TextNode {
	return &TextNode{NodeType: NodeText, Pos: pos, Text: []byte(text)}
}
func (n *TextNode) String() string {
	return fmt.Sprintf(textFormat, n.Text)
}
func (n *TextNode) writeTo(sb *strings.Builder) {
	sb.WriteString(n.String())
}

// LinkNode holds hyperlink.
type LinkNode struct {
	NodeType
	Pos
	URL      *url.URL
	Friendly string
	//todo does the gemini spec require exact spaces?
	//whtspace []string
	Text []byte // The original textual representation of the input.
}

func (t *Tree) newLink(pos Pos, text string) *LinkNode {
	return &LinkNode{NodeType: NodeLink, Pos: pos, Text: []byte(text)}
}
func (n LinkNode) String() string {
	return fmt.Sprintf("%s %s", n.URL, n.Friendly)
}
func (n LinkNode) writeTo(sb *strings.Builder) {
	sb.WriteString(n.String())
}

// BlankNode represents empty line.
type blankLine struct {
	whtspace string
}
