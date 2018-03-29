package transformer

import (
	"fmt"

	"gopkg.in/bblfsh/sdk.v1/uast"
	"gopkg.in/src-d/go-errors.v1"
)

type Transformer interface {
	Do(n uast.Node) (uast.Node, error)
}

type CodeTransformer interface {
	OnCode(code string) Transformer
}

type Sel interface {
	Check(st *State, n uast.Node) (bool, error)
}

type Mod interface {
	Construct(st *State, n uast.Node) (uast.Node, error)
}

type Op interface {
	Sel
	Mod
}

// Transformers appends all provided transformer slices into single one.
func Transformers(arr ...[]Transformer) []Transformer {
	var out []Transformer
	for _, a := range arr {
		out = append(out, a...)
	}
	return out
}

var _ Transformer = (TransformFunc)(nil)

type TransformFunc func(n uast.Node) (uast.Node, bool, error)

func (f TransformFunc) Do(n uast.Node) (uast.Node, error) {
	var last error
	nn, ok := uast.Apply(n, func(n uast.Node) (uast.Node, bool) {
		nn, ok, err := f(n)
		if err != nil {
			last = err
			return n, false
		} else if !ok {
			return n, false
		}
		return nn, ok
	})
	if ok {
		return nn, last
	}
	return n, last
}

var (
	ErrVariableRedeclared = errors.NewKind("variable %q redeclared (%v vs %v)")
	ErrVariableNotDefined = errors.NewKind("variable %q is not defined")
	ErrExpectedObject     = errors.NewKind("expected object, got %T")
	ErrExpectedList       = errors.NewKind("expected list, got %T")
	ErrExpectedValue      = errors.NewKind("expected value, got %T")
	ErrUnhandledValueIn   = errors.NewKind("unhandled value: %v in %v")
	ErrUnexpectedNode     = errors.NewKind("expected node to be nil, got: %v")
	ErrUnexpectedValue    = errors.NewKind("unexpected value: %v")
	ErrUnexpectedType     = errors.NewKind("unexpected type: %v")
	ErrAmbiguousValue     = errors.NewKind("map has ambiguous value %v")
	ErrFewSteps           = errors.NewKind("mapping should contains multiple steps")
	ErrUnusedField        = errors.NewKind("field was not used: %v")
	ErrDuplicateField     = errors.NewKind("duplicate field: %v")

	errAnd     = errors.NewKind("op %d (%T)")
	errKey     = errors.NewKind("key %q")
	errElem    = errors.NewKind("elem %d (%T)")
	errAppend  = errors.NewKind("append")
	errMapping = errors.NewKind("mapping %q")

	errCheck     = errors.NewKind("check")
	errConstruct = errors.NewKind("construct")
)

func Map(name string, src, dst Op) Mapping {
	return Mapping{Name: name, Steps: []Step{
		{Name: "src", Op: src},
		{Name: "dst", Op: dst},
	}}
}

var _ Transformer = Mapping{}

type Step struct {
	Name string
	Op   Op
}

type Mapping struct {
	Name  string
	Steps []Step
}

func (m Mapping) Reverse() Mapping {
	n := len(m.Steps)
	steps := make([]Step, n)
	for i, s := range m.Steps {
		steps[n-1-i] = s
	}
	m.Steps = steps
	return m
}

func applyMap(src, dst Op, n uast.Node) (uast.Node, error) {
	var errs []error
	_, objOp := src.(ObjectOp)
	_, arrOp := src.(ArrayOp)
	nn, ok := uast.Apply(n, func(n uast.Node) (uast.Node, bool) {
		if n != nil {
			if objOp {
				if _, ok := n.(uast.Object); !ok {
					return n, false
				}
			} else if arrOp {
				if _, ok := n.(uast.List); !ok {
					return n, false
				}
			}
		}
		st := NewState()
		if ok, err := src.Check(st, n); err != nil {
			errs = append(errs, errCheck.Wrap(err))
			return n, false
		} else if !ok {
			return n, false
		}
		nn, err := dst.Construct(st, nil)
		if err != nil {
			errs = append(errs, errConstruct.Wrap(err))
			return n, false
		}
		return nn, true
	})
	var first error
	if len(errs) != 0 {
		first = errs[0] // TODO: return multi-error
	}
	if ok {
		return nn, first
	}
	return n, first
}

func (m Mapping) Do(n uast.Node) (uast.Node, error) {
	if len(m.Steps) <= 1 {
		return n, ErrFewSteps.New()
	}
	steps := m.Steps
	var err error
	for len(steps) >= 2 {
		src, dst := steps[0], steps[1]
		n, err = applyMap(src.Op, dst.Op, n)
		if err != nil {
			return n, errMapping.Wrap(err, m.Name)
		}
		steps = steps[1:]
	}
	return n, err
}

// NewState creates a new state for Ops to work on.
// It stores variables, flags and anything that necessary
// for transformation steps to persist data.
func NewState() *State {
	return &State{}
}

type procObject struct {
	name   string
	fields uast.Object
}

type State struct {
	vars   map[string]uast.Node
	states map[string][]*State
	objs   []*procObject
}

func (st *State) Clone() *State {
	st2 := NewState()
	if len(st.vars) != 0 {
		st2.vars = make(map[string]uast.Node)
	}
	for k, v := range st.vars {
		st2.vars[k] = v
	}
	if len(st.states) != 0 {
		st2.states = make(map[string][]*State)
	}
	for k, v := range st.states {
		st2.states[k] = v
	}
	// TODO: clone each procObj?
	st2.objs = append([]*procObject{}, st.objs...)
	return st2
}

func (st *State) ApplyFrom(st2 *State) {
	if len(st2.vars) != 0 && st.vars == nil {
		st.vars = make(map[string]uast.Node)
	}
	for k, v := range st2.vars {
		if _, ok := st.vars[k]; !ok {
			st.vars[k] = v
		}
	}
	if len(st2.states) != 0 && st.states == nil {
		st.states = make(map[string][]*State)
	}
	for k, v := range st2.states {
		if _, ok := st.states[k]; !ok {
			st.states[k] = v
		}
	}
	if len(st2.objs) > len(st.objs) {
		st.objs = append(st.objs, st2.objs[len(st.objs):]...)
	}
}

func (st *State) GetVar(name string) (uast.Node, bool) {
	n, ok := st.vars[name]
	return n, ok
}

func (st *State) MustGetVar(name string) (uast.Node, error) {
	n, ok := st.GetVar(name)
	if !ok {
		return nil, ErrVariableNotDefined.New(name)
	}
	return n, nil
}

func (st *State) SetVar(name string, val uast.Node) error {
	cur, ok := st.vars[name]
	if !ok {
		// not declared
		if st.vars == nil {
			st.vars = make(map[string]uast.Node)
		}
		st.vars[name] = val
		return nil
	}
	v1, ok1 := cur.(uast.Value)
	v2, ok2 := val.(uast.Value)
	// the only exception is two primitive values that are equal
	if ok1 && ok2 && v1 == v2 {
		// already declared, and value is alredy in the map
		return nil
	}
	return ErrVariableRedeclared.New(name, cur, val)
}

func (st *State) GetStateVar(name string) ([]*State, bool) {
	n, ok := st.states[name]
	return n, ok
}

func (st *State) SetStateVar(name string, sub []*State) error {
	cur, ok := st.states[name]
	if ok {
		return ErrVariableRedeclared.New(name, cur, sub)
	}
	if st.states == nil {
		st.states = make(map[string][]*State)
	}
	st.states[name] = sub
	return nil
}

func (st *State) StartObject(name string) (func(), error) {
	obj := &procObject{
		name:   name,
		fields: make(uast.Object),
	}
	if name != "" {
		if err := st.SetVar(name, obj.fields); err != nil {
			return nil, err
		}
	}
	st.objs = append(st.objs, obj)
	return func() {
		if len(st.objs) == 0 {
			panic("no active objects on the stack")
		}
		i := len(st.objs) - 1
		cur := st.objs[i]
		st.objs = st.objs[:i]
		if cur.name != name {
			panic(fmt.Errorf("stack is broken: expected object %q, got %q", name, cur.name))
		}
	}, nil
}
func (st *State) UseKey(key string, val uast.Node) {
	if len(st.objs) == 0 {
		return
	}
	cur := st.objs[len(st.objs)-1]
	cur.fields[key] = val
}
