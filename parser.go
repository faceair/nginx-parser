package nginxparser

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"unicode"
)

type Directive struct {
	Line      int          `json:"line"`
	FileName  string       `json:"filename"`
	Directive string       `json:"directive"`
	Args      []string     `json:"args,omitempty"`
	Block     []*Directive `json:"block,omitempty"`
	Comment   string       `json:"comment,omitempty"`
}


func New(options *ParseOptions) *Parser {
	if options == nil {
		options = &ParseOptions{}
	}
	if options.Glob == nil {
		options.Glob = filepath.Glob
	}
	if options.Open == nil {
		options.Open = func(name string) (io.ReadCloser, error) {
			file, err := os.Open(name)
			return io.NopCloser(file), err
		}
	}
	return &Parser{options: options}
}

type ParseOptions struct {
	SingleFile bool
	Root       string
	Glob       func(pattern string) (matches []string, err error)
	Open       func(name string) (io.ReadCloser, error)
}

type Parser struct {
	options  *ParseOptions
	filename string
	line     int
}

func (p *Parser) ParseFile(filename string) ([]*Directive, error) {
	p.filename = filename
	file, err := p.options.Open(p.filename)
	if err != nil {
		return nil, err
	}
	return p.ParseReader(file)
}

func (p *Parser) ParseString(s string) ([]*Directive, error) {
	return p.ParseReader(bytes.NewReader([]byte(s)))
}

func (p *Parser) ParseReader(rd io.Reader) ([]*Directive, error) {
	reader := bufio.NewReader(rd)
	p.line = 1
	directives, err := p.parseReader(reader)
	if err != nil {
		return nil, err
	}
	for {
		b, err := reader.ReadByte()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if unicode.IsSpace(rune(b)) {
			continue
		}
		return nil, fmt.Errorf(`unexpected end in file %s line %d`, p.filename, p.line)
	}
	return directives, nil
}


const (
	stateScanDirective = "ScanDirective"
	stateScanArgs      = "ScanArgs"
)

func (p *Parser) parseReader(reader *bufio.Reader) ([]*Directive, error) {
	directives := make([]*Directive, 0)

	var buf bytes.Buffer
	var current *Directive
	var state string

readConfBlock:
	for {
		b, err := reader.ReadByte()
		if err == io.EOF {
			return directives, nil
		}

		if buf.Len() == 0 {
			switch b {
			case '#':
				comment, _, _ := reader.ReadLine()
				if current == nil {
					current = &Directive{
						Line:      p.line,
						FileName:  p.filename,
						Directive: "#",
						Args:      make([]string, 0),
						Block:     make([]*Directive, 0),
					}
				}
				p.line++
				if len(current.Comment) != 0 {
					current.Comment += " "
				}
				current.Comment += string(comment)
				if current.Directive == "#" {
					directives = append(directives, current)
					current = nil
				}
				continue
			case '/':
				unread, err := reader.Peek(1)
				if err != nil {
					return nil, err
				}
				if unread[0] == '/' {
					_, _ = reader.ReadByte()
					comment, _, _ := reader.ReadLine()
					if current == nil {
						current = &Directive{
							Line:      p.line,
							FileName:  p.filename,
							Directive: "#",
							Args:      make([]string, 0),
							Block:     make([]*Directive, 0),
						}
					}
					p.line++
					if len(current.Comment) != 0 {
						current.Comment += " "
					}
					current.Comment += string(comment)
					if current.Directive == "#" {
						directives = append(directives, current)
						current = nil
					}
				} else {
					buf.WriteByte('/')
				}
				continue
			}
		}

		switch b {
		case ' ', '\t':
			switch state {
			case stateScanDirective:
				if buf.Len() > 0 {
					if current == nil {
						current = &Directive{
							Line:      p.line,
							FileName:  p.filename,
							Directive: buf.String(),
							Args:      make([]string, 0),
							Block:     make([]*Directive, 0),
						}
					}
					buf.Reset()
					state = stateScanArgs
				}
			case stateScanArgs:
				if buf.Len() > 0 {
					current.Args = append(current.Args, buf.String())
					buf.Reset()
				}
			}
		case '\n':
			switch state {
			case stateScanDirective:
				if buf.Len() > 0 {
					if current == nil {
						current = &Directive{
							Line:      p.line,
							FileName:  p.filename,
							Directive: buf.String(),
							Args:      make([]string, 0),
							Block:     make([]*Directive, 0),
						}
					}
					buf.Reset()
					state = stateScanArgs
				}
				p.line++
			case stateScanArgs:
				p.line++
				if buf.Len() > 0 {
					current.Args = append(current.Args, buf.String())
					buf.Reset()
				}
			}
		case '\\':
			nb, err := reader.ReadByte()
			if err != nil {
				return nil, err
			}
			switch nb {
			case '"', '\'', '\\':
				b = nb
			case 'n':
				b = '\n'
			case 'r':
				b = '\r'
			case 't':
				b = '\t'
			default:
				b = nb
			}
			buf.WriteByte(b)
		case '"', '\'':
			if state == stateScanArgs && buf.Len() != 0 {
				buf.WriteByte(b)
				continue
			}

		readString:
			for {
				for {
					nr, _, err := reader.ReadRune()
					if err != nil {
						return nil, err
					}
					if nr == rune(b) {
						break
					}
					if nr == '\n' {
						p.line++
					}
					if err != nil {
						return nil, err
					}

					if nr == '\\' {
						nnb, err := reader.ReadByte()
						if err != nil {
							return nil, err
						}
						switch nnb {
						case '"', '\'', '\\':
							nr = rune(nnb)
						case 'n':
							nr = '\n'
						case 'r':
							nr = '\r'
						case 't':
							nr = '\t'
						default:
							buf.WriteRune(nr)
							nr = rune(nnb)
						}
					}
					buf.WriteRune(nr)
				}

				switch state {
				case stateScanDirective:
					current = &Directive{
						Line:      p.line,
						FileName:  p.filename,
						Directive: buf.String(),
						Args:      make([]string, 0),
						Block:     make([]*Directive, 0),
					}
					buf.Reset()
					state = stateScanArgs
					break readString
				case stateScanArgs:
					for i := 1; ; i++ {
						unread, err := reader.Peek(i)
						if err != nil {
							return nil, err
						}
						if unicode.IsSpace(rune(unread[i-1])) {
							continue
						}
						if unread[i-1] == b {
							for c := 0; c < i; c++ {
								nb, _ := reader.ReadByte()
								if nb == '\n' {
									p.line++
								}
							}
							continue readString
						}
						break
					}

					current.Args = append(current.Args, buf.String())
					buf.Reset()
					break readString
				}
			}
		case ';':
			switch state {
			case stateScanDirective:
				if buf.Len() > 0 {
					directives = append(directives, &Directive{
						Line:      p.line,
						FileName:  p.filename,
						Directive: buf.String(),
						Args:      make([]string, 0),
						Block:     make([]*Directive, 0),
					})
					current = nil
					buf.Reset()
				}
			case stateScanArgs:
				if buf.Len() > 0 {
					current.Args = append(current.Args, buf.String())
				}

				if !p.options.SingleFile && current.Directive == "include" {
					for _, arg := range current.Args {
						if !strings.HasPrefix(arg, "/") {
							if p.options.Root == "" {
								return nil, fmt.Errorf("not found `root` dir in options")
							}
							arg = path.Join(p.options.Root, arg)
						}
						filenames, err := p.options.Glob(arg)
						if err != nil {
							return nil, err
						}
						for _, filename := range filenames {
							blockDirectives, err := New(p.options).ParseFile(filename)
							if err != nil {
								return nil, err
							}
							current.Block = append(current.Block, blockDirectives...)
						}
					}
				}

				directives = append(directives, current)
				current = nil
				buf.Reset()
				state = stateScanDirective
			}
		case '{':
			switch state {
			case stateScanDirective:
				if buf.Len() == 0 {
					return nil, fmt.Errorf(`unexpected '%c' in file %s line %d`, b, p.filename, p.line)
				}

				current = &Directive{
					Line:      p.line,
					FileName:  p.filename,
					Directive: buf.String(),
					Args:      make([]string, 0),
				}
				current.Block, err = p.parseReader(reader)
				if err != nil {
					return nil, err
				}
				directives = append(directives, current)
				current = nil
				buf.Reset()
			case stateScanArgs:
				if buf.Len() > 0 {
					current.Args = append(current.Args, buf.String())
				}

				buf.Reset()
				if strings.HasSuffix(current.Directive, "_by_lua_block") {
					depth := 0
				readLuaBlock:
					for {
						b, err = reader.ReadByte()
						if err != nil {
							return nil, err
						}
						switch b {
						case '-':
							unread, err := reader.Peek(1)
							if err != nil {
								return nil, err
							}
							if unread[0] == '-' {
								buf.WriteByte(b)
								comment, _, err := reader.ReadLine()
								if err != nil {
									return nil, err
								}
								buf.WriteString(string(comment))
								buf.WriteByte('\n')
								p.line++
								continue
							}
						case '\n':
							p.line++
						case '"', '\'':
							buf.WriteByte(b)
							for {
								nr, _, err := reader.ReadRune()
								if err != nil {
									return nil, err
								}
								if nr == rune(b) {
									break
								}
								if nr == '\\' {
									buf.WriteRune(nr)
									nr, _, err = reader.ReadRune()
									if err != nil {
										return nil, err
									}
								}
								buf.WriteRune(nr)
							}
						case '{':
							depth++
						case '}':
							if depth != 0 {
								depth--
							} else {
								break readLuaBlock
							}
						}
						buf.WriteByte(b)
					}
					current.Args = append(current.Args, strings.TrimRightFunc(buf.String(), unicode.IsSpace))
				} else {
					if current.Directive == "if" {
						lastArgIndex := len(current.Args) - 1
						if len(current.Args) > 0 && strings.HasPrefix(current.Args[0], "(") && strings.HasSuffix(current.Args[lastArgIndex], ")") {
							current.Args[0] = strings.TrimLeftFunc(strings.TrimPrefix(current.Args[0], "("), unicode.IsSpace)
							current.Args[lastArgIndex] = strings.TrimRightFunc(strings.TrimSuffix(current.Args[lastArgIndex], ")"), unicode.IsSpace)
							if len(current.Args[0]) == 0 {
								current.Args = current.Args[1:]
								lastArgIndex -= 1
							}
							if len(current.Args[lastArgIndex]) == 0 {
								current.Args = current.Args[:lastArgIndex]
							}
						}
					}

					current.Block, err = p.parseReader(reader)
					if err != nil {
						return nil, err
					}
				}

				directives = append(directives, current)
				current = nil
				buf.Reset()
				state = stateScanDirective
			}
		case '}':
			switch state {
			case stateScanDirective:
				break readConfBlock
			case stateScanArgs:
				return nil, fmt.Errorf(`unexpected '%c' in file %s line %d`, b, p.filename, p.line)
			}
		case '$':
			buf.WriteByte(b)
			unread, err := reader.Peek(1)
			if err != nil {
				return nil, err
			}
			if unread[0] == '{' {
				for {
					nb, err := reader.ReadByte()
					if err != nil {
						return nil, err
					}
					buf.WriteByte(nb)
					if nb == '}' {
						break
					}
				}
			}
		case '\r':
		default:
			buf.WriteByte(b)
		}
	}
	return directives, nil
}
