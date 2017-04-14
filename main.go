package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mattn/go-runewidth"
	flag "github.com/ogier/pflag"
)

func widthANSI(s string) int {
	a := false
	w := 0
	for _, r := range s {
		if r == '\x1b' {
			a = true
		}
		if a {
			if r == 'm' {
				a = false
			}
			continue
		}
		c := runewidth.RuneWidth(r)
		if c != -1 {
			w += c
		}
	}
	return w
}

func escCont(s string) string {
	i := strings.LastIndex(s, "\x1b")
	if i == -1 {
		return ""
	}
	for j := i; j < len(s); j++ {
		if s[j] == 'm' {
			return s[i : j+1]
		}
	}
	return s[i:]
}

func say(w *bytes.Buffer, b borderStyle, lines []string) error {
	max := 0
	for _, l := range lines {
		w := widthANSI(l)
		if w > max {
			max = w
		}
	}

	ml := widthANSI(b[borderMiddle])
	maxp := max + ml*2

	w.WriteString(b[borderTopLeft])
	tl := widthANSI(b[borderTop])
	for i := 0; i < maxp; i += tl {
		w.WriteString(b[borderTop])
	}
	w.WriteString(b[borderTopRight])
	w.WriteString("\n")

	var esc string
	for _, line := range lines {
		// continue escapes
		w.WriteString(esc)

		w.WriteString(b[borderLeft])
		w.WriteString(b[borderMiddle])
		w.WriteString(line)
		l := widthANSI(line) - ml
		for i := 0; i < max-l; i += ml {
			w.WriteString(b[borderMiddle])
		}
		w.WriteString(b[borderRight])
		if esc != "" {
			w.WriteString("\x1b[0m")
		}
		w.WriteString("\n")

		esc = escCont(line)
	}

	w.WriteString(b[borderBottomLeft])
	bl := widthANSI(b[borderBottom])
	for i := 0; i < maxp; i += bl {
		w.WriteString(b[borderBottom])
	}
	w.WriteString(b[borderBottomRight])

	return nil
}

func tewi(w *bytes.Buffer, cow []string, eyes, tongue, line string) error {
	cowp := make([]string, 0, len(cow))
	for _, c := range cow {
		if strings.HasPrefix(c, "$the_cow") ||
			strings.HasPrefix(c, "#") ||
			strings.HasPrefix(c, "EOC") {
			continue
		}
		cowp = append(cowp, c)
	}

	r := strings.NewReplacer(
		"$thoughts", line,
		"\\\\", "\\",
		"\\@", "@",
		"eyes", eyes,
		"tongue", tongue,
	)

	for _, c := range cowp {
		w.WriteString("\n")
		_, err := r.WriteString(w, c)
		if err != nil {
			return err
		}
	}

	return nil
}

func cowPath() []string {
	p := []string{}
	s := os.Getenv("COWPATH")
	if s != "" {
		p = append(p, strings.Split(s, ":")...)
	} else {
		p = append(p, // TODO: don't hardcode this
			filepath.Join(os.Getenv("HOME"), ".cows"),
			"/usr/share/cows")
	}
	return p
}

func readCow(name string) (string, error) {
	// probably absolute path
	if strings.Contains(name, "/") || filepath.Ext(name) == ".cow" {
		b, err := ioutil.ReadFile(name)
		return string(b), err
	}

	name = name + ".cow"
	for _, cp := range cowPath() {
		p := filepath.Join(cp, name)
		b, err := ioutil.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		return string(b), err
	}
	return "", fmt.Errorf("could not find cowfile: %s", name)
}

func listCows() ([]string, error) {
	l := []string{}
	for _, cp := range cowPath() {
		d, err := ioutil.ReadDir(cp)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, f := range d {
			if filepath.Ext(f.Name()) != ".cow" {
				continue
			}
			l = append(l, strings.TrimSuffix(f.Name(), ".cow"))
		}
	}
	return l, nil
}

func handle(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err)
		os.Exit(1)
	}
}

func main() {
	defBorder := "unicode"
	if path.Base(os.Args[0]) == "tewithink" {
		defBorder = "think"
	}

	var (
		border = flag.StringP("border", "b", defBorder,
			"which border to use (try list, preview)")
		eyes   = flag.StringP("eyes", "e", "oo", "change eyes")
		tongue = flag.StringP("tongue", "t", "  ", "change tongue")
		list   = flag.BoolP("list", "l", false, "list cowfiles")
		file   = flag.StringP("file", "f", "tes", "cowfile")
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [option ...] [text]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(0)
	}

	flag.Parse()

	if *list {
		l, err := listCows()
		handle(err)
		fmt.Println(strings.Join(l, " "))
		return
	}

	if *border == "list" || *border == "preview" {
		l := []string{}
		for k := range borderStyles {
			l = append(l, k)
		}
		sort.Strings(l)
		if *border == "list" {
			fmt.Println(strings.Join(l, " "))
			return
		}
		w := &bytes.Buffer{}
		for _, b := range l {
			handle(say(w, borderStyles[b], []string{b}))
			fmt.Printf("%s\n    %s\n", w.String(), borderStyles[b][borderLine])
			w.Reset()
		}
		return
	}

	b, ok := borderStyles[*border]
	if !ok {
		handle(fmt.Errorf("no such border style: %s", *border))
	}

	cow, err := readCow(*file)
	handle(err)

	args := flag.Args()
	var lines []string
	if len(args) != 0 {
		lines = strings.Split(strings.Replace(strings.Join(args, " "),"\t", "   ", -1), "\n")
	} else {
		b, err := ioutil.ReadAll(os.Stdin)
		handle(err)
		b = bytes.TrimSuffix(b, []byte("\n"))
		lines = strings.Split(strings.Replace(string(b), "\t", "   ", -1), "\n")
	}

	w := &bytes.Buffer{}
	handle(say(w, b, lines))
	handle(tewi(w, strings.Split(cow, "\n"), *eyes, *tongue, b[borderLine]))
	w.WriteTo(os.Stdout)
}
