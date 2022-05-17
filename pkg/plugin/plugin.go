package plugin

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/console"
	"github.com/dop251/goja_nodejs/require"
	kuberdepv1 "github.com/joelanford/kuberdep/api/v1"
	"github.com/joelanford/kuberdep/pkg/plugin/functions"
	"hash/fnv"
	"path/filepath"
	"reflect"
	"runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

type Plugins interface {
	Exec(typ string, input Input) ([]kuberdepv1.Constraint, error)
}

type Input struct {
	Config  interface{}
	Subject Entity
	Items   []Entity
}

type plugins struct {
	vm *goja.Runtime
}

func New(ctx context.Context, cl client.Client) (Plugins, error) {
	vm := goja.New()
	registry := require.NewRegistry()
	registry.Enable(vm)
	console.Enable(vm)

	fmt.Println(len(functions.Functions))
	for _, f := range functions.Functions {
		name := filepath.Base(runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name())
		spl := strings.Split(name, ".")

		if len(spl) == 2 {
			name = spl[1]
		}
		vm.Set(name, f)
	}

	defs := &kuberdepv1.ConstraintDefinitionList{}
	if err := cl.List(ctx, defs); err != nil {
		return nil, err
	}

	for _, def := range defs.Items {
		idHash := "f" + hex.EncodeToString(fnv.New64().Sum([]byte(def.Spec.ID)))
		f := funcTemplate(idHash, def.Spec.Body)
		if _, err := vm.RunScript(fmt.Sprintf("%s.js", def.Spec.ID), f); err != nil {
			return nil, err
		}
	}
	return &plugins{vm}, nil
}

func (p *plugins) Exec(typ string, input Input) ([]kuberdepv1.Constraint, error) {
	idHash := "f" + hex.EncodeToString(fnv.New64().Sum([]byte(typ)))

	f, ok := goja.AssertFunction(p.vm.Get(idHash))
	if !ok {
		return nil, fmt.Errorf("failed to execute plugin for type %q: plugin function is undefined", typ)
	}

	val, err := f(goja.Undefined(), p.vm.ToValue(input))
	if err != nil {
		return nil, fmt.Errorf("failed to execute plugin for type %q: %v", typ, err)
	}

	res := val.Export()
	valJSON, err := json.Marshal(res)
	if err != nil {
		return nil, err
	}

	constraintsDecoder := json.NewDecoder(bytes.NewReader(valJSON))
	constraintsDecoder.DisallowUnknownFields()
	var constraints []kuberdepv1.Constraint
	if err := constraintsDecoder.Decode(&constraints); err != nil {
		return nil, err
	}
	return constraints, nil
}

func funcTemplate(name, body string) string {
	return fmt.Sprintf(`function %s(input){
	%s
}`, name, body)
}

type Entity struct {
	ID          string
	Object      interface{}
	Properties  []TypeValue
	Constraints []TypeValue
}

type TypeValue struct {
	Type  string
	Value interface{}
}

func ConvertEntities(entities ...kuberdepv1.Entity) ([]Entity, error) {
	var out []Entity
	for _, entity := range entities {
		obj, err := kuberdepv1.ValueToInterface(entity.Spec.Object.Raw)
		if err != nil {
			return nil, err
		}
		var props []TypeValue
		for _, p := range entity.Spec.Properties {
			pv, err := kuberdepv1.ValueToInterface(p.Value)
			if err != nil {
				return nil, err
			}
			props = append(props, TypeValue{Type: p.Type, Value: pv})
		}
		var constraints []TypeValue
		for _, c := range entity.Spec.Constraints {
			cv, err := kuberdepv1.ValueToInterface(c.Value)
			if err != nil {
				return nil, err
			}
			constraints = append(constraints, TypeValue{Type: c.Type, Value: cv})
		}
		out = append(out, Entity{
			ID:          entity.Name,
			Object:      obj,
			Properties:  props,
			Constraints: constraints,
		})
	}
	return out, nil
}
