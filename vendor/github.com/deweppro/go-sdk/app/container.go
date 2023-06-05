package app

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/deweppro/go-algorithms/graph/kahn"
	"github.com/deweppro/go-sdk/errors"
)

type _dic struct {
	kahn *kahn.Graph
	srv  *_serv
	list *dicMap
}

func newDic(ctx Context) *_dic {
	return &_dic{
		kahn: kahn.New(),
		srv:  newService(ctx),
		list: newDicMap(),
	}
}

// Down - stop all services in dependencies
func (v *_dic) Down() error {
	return v.srv.Down()
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

// Register - register a new dependency
func (v *_dic) Register(items ...interface{}) error {
	if v.srv.IsUp() {
		return errDepBuilderNotRunning
	}

	for _, item := range items {
		ref := reflect.TypeOf(item)
		switch ref.Kind() {

		case reflect.Struct:
			if err := v.list.Add(item, item, typeExist); err != nil {
				return err
			}

		case reflect.Func:
			for i := 0; i < ref.NumIn(); i++ {
				in := ref.In(i)
				if in.Kind() == reflect.Struct {
					if err := v.list.Add(in, reflect.New(in).Elem().Interface(), typeNewIfNotExist); err != nil {
						return err
					}
				}

			}
			if ref.NumOut() == 0 {
				if err := v.list.Add(ref, item, typeNew); err != nil {
					return err
				}
				continue
			}
			for i := 0; i < ref.NumOut(); i++ {
				if err := v.list.Add(ref.Out(i), item, typeNew); err != nil {
					return err
				}
			}

		default:
			if err := v.list.Add(item, item, typeExist); err != nil {
				return err
			}
		}
	}

	return nil
}

// Build - initialize dependencies
func (v *_dic) Build() error {
	if err := v.srv.MakeAsUp(); err != nil {
		return err
	}

	err := v.list.foreach(v.calcFunc, v.calcStruct, v.calcOther)
	if err != nil {
		return errors.Wrapf(err, "building dependency graph")
	}

	if err = v.kahn.Build(); err != nil {
		return errors.Wrapf(err, "dependency graph calculation")
	}

	return v.exec(nil)
}

// Inject - obtained dependence
func (v *_dic) Inject(item interface{}) error {
	_, err := v.callArgs(item)
	return err
}

// Invoke - obtained dependence
func (v *_dic) Invoke(item interface{}) error {
	if err := v.Register(item); err != nil {
		return err
	}

	if err := v.srv.MakeAsUp(); err != nil {
		return err
	}

	err := v.list.foreach(v.calcFunc, v.calcStruct, v.calcOther)
	if err != nil {
		return errors.Wrapf(err, "building dependency graph")
	}

	if err = v.kahn.Build(); err != nil {
		return errors.Wrapf(err, "dependency graph calculation")
	}

	ref := reflect.TypeOf(item)
	addr, ok := getRefAddr(ref)
	if !ok {
		return fmt.Errorf("resolve invoke reference")
	}

	return v.exec(&addr)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

var empty = "EMPTY"

func (v *_dic) calcFunc(outAddr string, outRef reflect.Type) error {
	if outRef.NumIn() == 0 {
		if err := v.kahn.Add(empty, outAddr); err != nil {
			return errors.Wrapf(err, "cant add [->%s] to graph", outAddr)
		}
	}

	for i := 0; i < outRef.NumIn(); i++ {
		inRef := outRef.In(i)
		inAddr, _ := getRefAddr(inRef)

		if _, err := v.list.Get(inAddr); err != nil {
			return errors.Wrapf(err, "cant add [%s->%s] to graph", inAddr, outAddr)
		}
		if err := v.kahn.Add(inAddr, outAddr); err != nil {
			return errors.Wrapf(err, "cant add [%s->%s] to graph", inAddr, outAddr)
		}
	}

	return nil
}

func (v *_dic) calcStruct(outAddr string, outRef reflect.Type) error {
	if outRef.NumField() == 0 {
		if err := v.kahn.Add(empty, outAddr); err != nil {
			return errors.Wrapf(err, "cant add [->%s] to graph", outAddr)
		}
		return nil
	}
	for i := 0; i < outRef.NumField(); i++ {
		inRef := outRef.Field(i).Type
		inAddr, _ := getRefAddr(inRef)

		if _, err := v.list.Get(inAddr); err != nil {
			return errors.Wrapf(err, "cant add [%s->%s] to graph", inAddr, outAddr)
		}
		if err := v.kahn.Add(inAddr, outAddr); err != nil {
			return errors.Wrapf(err, "cant add [%s->%s] to graph", inAddr, outAddr)
		}
	}
	return nil
}

func (v *_dic) calcOther(_ string, _ reflect.Type) error {
	return nil
}

func (v *_dic) callFunc(item interface{}) ([]reflect.Value, error) {
	ref := reflect.TypeOf(item)
	args := make([]reflect.Value, 0, ref.NumIn())

	for i := 0; i < ref.NumIn(); i++ {
		inRef := ref.In(i)
		inAddr, _ := getRefAddr(inRef)
		vv, err := v.list.Get(inAddr)
		if err != nil {
			return nil, err
		}
		args = append(args, reflect.ValueOf(vv))
	}

	args = reflect.ValueOf(item).Call(args)
	for _, arg := range args {
		if err, ok := arg.Interface().(error); ok && err != nil {
			return nil, err
		}
	}

	return args, nil
}

func (v *_dic) callStruct(item interface{}) ([]reflect.Value, error) {
	ref := reflect.TypeOf(item)
	value := reflect.New(ref)
	args := make([]reflect.Value, 0, ref.NumField())

	for i := 0; i < ref.NumField(); i++ {
		inRef := ref.Field(i)
		inAddr, _ := getRefAddr(inRef.Type)
		vv, err := v.list.Get(inAddr)
		if err != nil {
			return nil, err
		}
		value.Elem().FieldByName(inRef.Name).Set(reflect.ValueOf(vv))
	}

	return append(args, value.Elem()), nil
}

func (v *_dic) callArgs(item interface{}) ([]reflect.Value, error) {
	ref := reflect.TypeOf(item)

	switch ref.Kind() {
	case reflect.Func:
		return v.callFunc(item)
	case reflect.Struct:
		return v.callStruct(item)
	default:
		return []reflect.Value{reflect.ValueOf(item)}, nil
	}
}

func (v *_dic) exec(breakPoint *string) error {
	names := make(map[string]struct{})
	for _, name := range v.kahn.Result() {
		if name == empty {
			continue
		}
		names[name] = struct{}{}
	}

	for _, name := range v.kahn.Result() {
		if _, ok := names[name]; !ok {
			continue
		}
		if v.list.HasType(name, typeExist) {
			continue
		}

		item, err := v.list.Get(name)
		if err != nil {
			return err
		}

		args, err := v.callArgs(item)
		if err != nil {
			return errors.Wrapf(err, "initialize error [%s]", name)
		}

		for _, arg := range args {
			addr, _ := getRefAddr(arg.Type())
			if vv, ok := asService(arg); ok {
				if err = v.srv.AddAndRun(vv); err != nil {
					return errors.Wrapf(err, "service initialization error [%s]", addr)
				}
			}
			if vv, ok := asServiceContext(arg); ok {
				if err = v.srv.AddAndRun(vv); err != nil {
					return errors.Wrapf(err, "service initialization error [%s]", addr)
				}
			}
			delete(names, addr)
			if arg.Type().String() == "error" {
				continue
			}
			if err = v.list.Add(arg.Type(), arg.Interface(), typeExist); err != nil {
				return errors.Wrapf(err, "initialize error [%s]", addr)
			}
		}
		delete(names, name)

		if breakPoint != nil && *breakPoint == name {
			break
		}
	}

	v.srv.IterateOver()

	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

const (
	typeNew int = iota
	typeNewIfNotExist
	typeExist
)

type (
	dicMapItem struct {
		Value interface{}
		Type  int
	}
	dicMap struct {
		data map[string]*dicMapItem
		mux  sync.RWMutex
	}
)

func newDicMap() *dicMap {
	return &dicMap{
		data: make(map[string]*dicMapItem),
	}
}

func (v *dicMap) Add(place, value interface{}, t int) error {
	v.mux.Lock()
	defer v.mux.Unlock()

	ref, ok := place.(reflect.Type)
	if !ok {
		ref = reflect.TypeOf(place)
	}

	addr, ok := getRefAddr(ref)
	if !ok {
		if addr != "error" {
			return fmt.Errorf("dependency [%s] is not supported", addr)
		}
		//return nil
	}

	if vv, ok := v.data[addr]; ok {
		if t == typeNewIfNotExist {
			return nil
		}
		if vv.Type == typeExist {
			return fmt.Errorf("dependency [%s] already initiated", addr)
		}
	}
	v.data[addr] = &dicMapItem{
		Value: value,
		Type:  t,
	}

	return nil
}

func (v *dicMap) Get(addr string) (interface{}, error) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	if vv, ok := v.data[addr]; ok {
		return vv.Value, nil
	}
	return nil, fmt.Errorf("dependency [%s] not initiated", addr)
}

func (v *dicMap) HasType(addr string, t int) bool {
	v.mux.RLock()
	defer v.mux.RUnlock()

	if vv, ok := v.data[addr]; ok {
		return vv.Type == t
	}
	return false
}

func (v *dicMap) Step(addr string) (int, error) {
	v.mux.RLock()
	defer v.mux.RUnlock()

	if vv, ok := v.data[addr]; ok {
		return vv.Type, nil
	}
	return 0, fmt.Errorf("dependency [%s] not initiated", addr)
}

func (v *dicMap) foreach(kFunc, kStruct, kOther func(addr string, ref reflect.Type) error) error {
	v.mux.RLock()
	defer v.mux.RUnlock()

	for addr, item := range v.data {
		if item.Type == typeExist {
			continue
		}

		ref := reflect.TypeOf(item.Value)
		var err error
		switch ref.Kind() {
		case reflect.Func:
			err = kFunc(addr, ref)
		case reflect.Struct:
			err = kStruct(addr, ref)
		default:
			err = kOther(addr, ref)
		}

		if err != nil {
			return err
		}
	}
	return nil
}
