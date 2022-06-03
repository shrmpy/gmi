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
		errp error
		abs  string
		dir  = flag.String("dir", "", "Work directory with Gemtext files")
	)
	flag.Parse()
	if abs, errp = filepath.Abs(*dir); errp != nil {
		log.Fatalf("Unknown path, %v", errp)
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
		var (
			errs  error
			rdr   *bufio.Reader
			ifile *os.File
			outf  string
			md    string
		)
		if ifile, errs = os.Open(path); errs != nil {
			log.Printf("ERROR file, %v", errs)
			return errs
		}
		defer ifile.Close()
		log.Printf("INFO reading, %s", d.Name())
		rdr = bufio.NewReader(ifile)

		if md, errs = ctrl.Retrieve(rdr); errs != nil {
			log.Printf("ERROR Retrieve, %v", errs)
			return errs
		}
		outf = fmt.Sprintf("%s.md", strings.TrimSuffix(path, ".gmi"))
		log.Printf("INFO writing, %s : %d", outf, len(md))
		errs = os.WriteFile(outf, []byte(md), d.Type())
		if errs != nil {
			log.Printf("ERROR output file, %v", errs)
			return errs
		}

		return nil
	})
}
func rewriteLink(n gmi.Node) string {
	var (
		lnk  = n.(*gmi.LinkNode)
		name = lnk.Friendly
		lu   = lnk.URL
	)
	if name == "" {
		name = lu.String()
	}
	// markdown of hyperlink
	return fmt.Sprintf("[=> %s](%s)\n", name, lu)
}
func rewritePlain(n gmi.Node) string {
	return fmt.Sprintf("%s\n", n)
}
