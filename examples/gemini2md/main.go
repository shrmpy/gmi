package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)
import "github.com/shrmpy/gmi"

func main() {
	var (
		aer error
		abs string
		dir = flag.String("dir", "", "Work directory with Gem files")
	)
	flag.Parse()

	if abs, aer = filepath.Abs(*dir); aer != nil {
		log.Fatalf("Unknown path, %v", aer)
	}

	var ctrl = gmi.NewControl(context.Background())
	ctrl.Attach(gmi.LinkLine, rewriteLink)
	ctrl.Attach(gmi.PlainLine, rewritePlain)
	filepath.WalkDir(abs, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("INFO walk halted, %v", err)
			return err
		}
		if d.IsDir() {
			log.Printf("INFO dir, %s", d.Name())
			return nil
		}
		if strings.ToLower(filepath.Ext(path)) != ".gmi" {
			log.Printf("INFO non .gmi skipped, %s", d.Name())
			return nil
		}

		file, fer := os.Open(path)
		if fer != nil {
			log.Printf("ERROR file, %v", fer)
			return fer
		}
		defer file.Close()
		log.Printf("INFO reading, %v", d.Name())
		var rdr = bufio.NewReader(file)

		md, wer := ctrl.Retrieve(rdr)
		if wer != nil {
			log.Printf("ERROR Retrieve, %v", wer)
			return wer
		}

		var ofile = fmt.Sprintf("%s.md", strings.TrimSuffix(path, ".gmi"))
		log.Printf("INFO writing, %s : %d", ofile, len(md))
		var oerr = os.WriteFile(ofile, []byte(md), d.Type())
		if oerr != nil {
			log.Printf("ERROR output file, %v", oerr)
			return oerr
		}

		return nil
	})
}
func rewriteLink(n gmi.Node) string {
	var lnk = n.(*gmi.LinkNode)
	var name = lnk.Friendly
	var lu = lnk.URL
	if name == "" {
		name = lu.String()
	}
	// markdown of hyperlink
	return fmt.Sprintf("[=> %s](%s)\n", name, lu)
}
func rewritePlain(n gmi.Node) string {
	return fmt.Sprintf("%s\n", n)
}
