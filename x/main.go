package main

import (
	"fmt"
	"os"
	"strings"
)

type SE interface {
	IsS() bool
	IsA() bool
	IsP() bool
	AsS() string
	AsF() float64
}

type se struct {
	isExpression, isAtom, isString bool
	value                          interface{}
}

type Atom struct {
	se
}

type Pair struct {
	se
	pcar, pcdr SE
}

type Proc struct {
	se
	isBuiltin bool
}

type Lisp struct {
	process *process
	Env     *Env
}

type Env struct {
	dict  map[string]SE
	outer *Env
}

type DefinedProc struct {
	params Pair
	body   SE
	env    *Env
}

type BuiltinProc = func(*process, *Env, []SE) (SE, error)

type process struct {
	pid string
}

type clause struct {
	pattern, template pattern
	ellipsis          map[string]int
}

type pattern struct {
	isVariable, isUnderscore, isLiteral, isList, hasEllipsis bool
	content                                                  SE
	listContent                                              []pattern
}

func main() {
}

func (s se) IsS() bool {
	return s.isAtom && s.isString
}

func (s se) IsA() bool {
	return s.isExpression && s.isAtom
}

func (s se) IsP() bool {
	return s.isExpression && !s.isAtom
}

func (s se) AsS() string {
	return s.value.(string)
}

func (s se) AsF() float64 {
	return s.value.(float64)
}

func Newstring(s string) Atom {
	a := NewAtom(s)
	a.isString = true
	return a
}

func NewAtom(v interface{}) Atom {
	return Atom{se{
		isExpression: true,
		isAtom:       true,
		value:        v,
	}}
}

func NP(car, cdr SE) Pair {
	return Pair{
		se: se{
			isExpression: true,
		},
		pcar: car,
		pcdr: cdr,
	}
}

func list2cons(list ...SE) Pair {
	if len(list) == 0 {
		return NP(nil, nil)
	}
	if len(list) == 1 {
		return NP(list[0], NP(nil, nil))
	}
	cons := NP(nil, nil)
	for i := len(list) - 1; i >= 0; i-- {
		cons = NP(list[i], cons)
	}
	return cons
}

func cons2list(p Pair) []SE {
	list := []SE{}
	for p != NP(nil, nil) {
		list = append(list, p.pcar)
		p = p.pcdr.(Pair)
	}
	return list
}

func add(p *process, env *Env, args []SE) (SE, error) {
	return NewAtom(args[0].AsF() + args[1].AsF()), nil
}

const ellipsis = "..."
const underscore = "_"

func init() {
	s := mustParse(`(syntax-rules ()
                                 ((_ ((var exp) ...) body1 body2 ...)
                                   ((lambda (var ...) (begin body1 body2 ...)) exp ...)))`)
	s1 := s.(Pair)
	syntaxRules("let", s1)
}

func mustParse(program string) SE {
	p, _ := parse(program)
	return p
}

func parse(program string) (SE, error) {
	p, _, err := readFromTokens(tokenize(program))
	return p, err
}

func expandMacro(p Pair) (SE, bool) {
	return p, false
}

func syntaxRules(keyword string, sr Pair) {
	literals := []string{keyword, "lambda", "define", "begin"}
	for _, e := range cons2list(sr.pcdr.(Pair).pcar.(Pair)) {
		literals = append(literals, e.AsS())
	}
	prepareClauses(sr, literals)
}

func prepareClauses(sr Pair, literals []string) {
	b := []bool{}
	for _, c := range cons2list(sr.pcdr.(Pair).pcdr.(Pair)) {
		cp := c.(Pair)
		s := map[string]string{}
		e := map[string]int{}
		p := analysePattern(literals, cp.pcar, s, e)
		t := analyseTemplate(literals, cp.pcdr.(Pair).pcar, s, e)
		c1 := clause{pattern: p, template: t, ellipsis: e}
		println("c1.pattern.isList: ", c1.pattern.isList)
		b = append(b, c1.pattern.isList)
		println("b[0]: ", b[0])
	}
	os.Exit(0)
}

var symbolCounter int

func gensym() string {
	symbolCounter += 1
	return string(fmt.Sprintf("gensym%d", symbolCounter))
}

func analyse(l []string, p SE, g map[string]string, b bool) pattern {
	if p.IsS() {
		s := p.AsS()
		if s == underscore {
			return pattern{isUnderscore: true}
		}
		for _, lt := range l {
			if lt == s {
				return pattern{isLiteral: true, content: p}
			}
		}
		if b {
			ns := gensym()
			g[s] = ns
			return pattern{isVariable: true, content: Newstring(ns)}
		}
		if ns, ok := g[s]; ok {
			return pattern{isVariable: true, content: Newstring(ns)}
		}
		return pattern{isVariable: true, content: p}
	}
	lc := []pattern{}
	list := cons2list(p.(Pair))
	for i := 0; i < len(list); i++ {
		pi := analyse(l, list[i], g, b)
		if i != len(list)-1 {
			sj := list[i+1]
			if sj.IsS() && sj.AsS() == ellipsis {
				pi.hasEllipsis = true
				i++
			}
		}
		lc = append(lc, pi)
	}
	return pattern{isList: true, listContent: lc}
}

func analysePattern(l []string, p SE, g map[string]string, e map[string]int) pattern {
	pt := analyse(l, p, g, true)
	analyseEllipsis(pt, e, 0)
	return pt
}

func analyseTemplate(l []string, t SE, g map[string]string, e map[string]int) pattern {
	return analyse(l, t, g, false)
}

func analyseEllipsis(p pattern, e map[string]int, d int) {
	if p.isVariable && (d != 0 || p.hasEllipsis) {
		ps := p.content.AsS()
		if p.hasEllipsis {
			d++
		}
		e[ps] = d
	} else if p.isList {
		nd := d
		if p.hasEllipsis {
			nd++
		}
		for _, pp := range p.listContent {
			analyseEllipsis(pp, e, nd)
		}
	}
}

func Multiparse(f string) ([]SE, error) {
	var e SE
	t := tokenize(f)
	exprs := []SE{}
	for len(t) > 0 {
		e, t, _ = readFromTokens(t)
		exprs = append(exprs, e)
	}
	return exprs, nil
}

func tokenize(s string) []string {
	return strings.Fields(strings.ReplaceAll(strings.ReplaceAll(s, "(", " ( "), ")", " ) "))
}

func readFromTokens(t []string) (SE, []string, error) {
	t0 := t[0]
	t = t[1:]
	switch t0 {
	case "(":
		list := []SE{}
		for t[0] != ")" {
			parsed, t1, _ := readFromTokens(t)
			t = t1
			list = append(list, parsed)
		}
		return list2cons(list...), t[1:], nil
	default:
		return atom(t0), t, nil
	}
}

func atom(t string) SE {
	if t == "0" || t == "1" {
		return NewAtom(float64(t[0] - '0'))
	}
	if t[0] == '\'' {
		q, _, _ := readFromTokens([]string{"(", "quote", t[1:], ")"})
		return q
	}
	return Newstring(t)
}
