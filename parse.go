package gmi

import (
	"fmt"
	"net/url"
)

type Tree struct {
	lex       *lexer
	pretoggle bool
	Root      *ListNode // top-level node of our tree
	peekCount int
	token     [3]item
}

// convenience entry point to parse the GEMtext
func Parse(text string) (*Tree, error) {
	t := &Tree{}
	// initiate the lexer defined separately by our package
	t.lex = lex("DEBUG-DEBUG", text)
	err := t.Parse()
	return t, err
}

// Parse accepts GEMtext and digests into structured (hier/tree) data
func (t *Tree) Parse() error {
	t.Root = t.newList(t.peek().pos)

	for t.peek().typ != itemEOF {
		switch n := t.textOrLink(); n.Type() {

		default:
			t.Root.append(n)
		}
	}
	return nil
}

//TODO core defines [text|blank|link|pre] lines
func (t *Tree) textOrLink() Node {
	switch token := t.next(); token.typ {
	case itemText:
		return t.newText(token.pos, token.val)
	case itemLink:
		return link(t, token)
	default:
		// only support subset for now
		panic(fmt.Errorf("unexpected %s in input", token))
	}
	return nil
}

// next returns the next token.
func (t *Tree) next() item {
	if t.peekCount > 0 {
		t.peekCount--
	} else {
		t.token[0] = t.lex.nextItem()
	}
	return t.token[t.peekCount]
}

// peek returns but does not consume the next token.
func (t *Tree) peek() item {
	if t.peekCount > 0 {
		return t.token[t.peekCount-1]
	}
	t.peekCount = 1
	t.token[0] = t.lex.nextItem()
	return t.token[0]
}

// construct link node from 2/3 tokens
func link(t *Tree, token item) Node {
	n := t.newLink(token.pos, token.val)
	var (
		err error
		it  item
	)
	//url
	it = t.next()
	if it.typ != itemLinkURL {
		panic(fmt.Errorf("problem with link input %s ", token))
	}

	n.URL, err = url.Parse(it.val)
	if err != nil {
		panic(fmt.Errorf("problem with link URL %s ", it))
	}
	//friendly description is optional
	it = t.peek()
	if it.typ == itemLinkDesc {
		t.next()
		n.Friendly = it.val
	}

	return n
}
