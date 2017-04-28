package usdt

/*
#include <stdlib.h>
#include "usdt.h"
*/
import "C"
import (
	"fmt"
	"reflect"
	"unsafe"
)

// Provider represents a DTrace USDT provider, currently implemented by wrapping
// the C library libusdt for use with Go.
type Provider struct {
	Name   string
	Module string
	Probes []*Probe

	cProvider *C.usdt_provider_t
}

// NewProvider constructs a Provider with the given Name and Module, which represent
// the first two elements of the four-tuple identifier for probes contained within.
func NewProvider(name string, module string) (*Provider, error) {
	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cModule := C.CString(module)
	defer C.free(unsafe.Pointer(cModule))

	cProvider := C.usdt_create_provider(cName, cModule)
	if cProvider == nil {
		return nil, fmt.Errorf("usdt_create_provider (%s:%s)", name, module)
	}

	return &Provider{
		Name:      name,
		Module:    module,
		Probes:    []*Probe{},
		cProvider: cProvider,
	}, nil
}

// Close frees the resources associated with the underlying C implementation of a provider.
func (p *Provider) Close() {
	if p.cProvider == nil {
		return
	}

	C.usdt_provider_free(p.cProvider)
	p.cProvider = nil
}

// Enable enables the underlying provider.
func (p *Provider) Enable() error {
	if p.cProvider == nil {
		return fmt.Errorf("DTrace [%s:%s]: provider closed", p.Name, p.Module)
	}

	ret := C.usdt_provider_enable(p.cProvider)
	if ret == 0 {
		return nil
	}

	return p.error()
}

// AddProbe adds the given probe to a provider.
func (p *Provider) AddProbe(probe *Probe) error {
	if probe == nil {
		return fmt.Errorf("probe may not be nil")
	}

	ret := C.usdt_provider_add_probe(p.cProvider, probe.cProbeDef)
	if ret == 0 {
		return nil
	}

	return p.error()
}

// Enabled returns true if the underlying provider is enabled, and false otherwise.
func (p *Provider) Enabled() bool {
	if p.cProvider == nil {
		return false
	}

	return C.int(p.cProvider.enabled) != 0
}

func (p *Provider) error() error {
	if p.cProvider == nil {
		return fmt.Errorf("DTrace [%s:%s]: provider closed", p.Name, p.Module)
	}

	message := C.GoString(p.cProvider.error)
	if len(message) == 0 {
		return nil
	}

	return fmt.Errorf("DTrace [%s:%s]: %s", p.Name, p.Module, message)
}

// Probe represents a DTrace USDT probe, wrapping the C implementation for use
// with Go.
type Probe struct {
	Function string
	Name     string

	cProbeDef *C.usdt_probedef_t
}

// NewProbe constructs a new probe definition for the given function and probe
// name, with arguments based on the C-converted Go types.
func NewProbe(function string, name string, argumentTypes ...reflect.Kind) (*Probe, error) {
	cFunction := C.CString(function)
	defer C.free(unsafe.Pointer(cFunction))

	cName := C.CString(name)
	defer C.free(unsafe.Pointer(cName))

	cArgumentTypes := make([]*C.char, len(argumentTypes)+1)
	for i, kind := range argumentTypes {
		switch kind {
		case reflect.Int:
			argType := C.CString("int")
			defer C.free(unsafe.Pointer(argType))
			cArgumentTypes[i] = argType
		case reflect.String:
			argType := C.CString("char *")
			defer C.free(unsafe.Pointer(argType))
			cArgumentTypes[i] = argType
		default:
			return nil, fmt.Errorf("Probe arguments may only be of kind Int or String")
		}
	}

	cProbeDef := C.usdt_create_probe(cFunction, cName, C.size_t(len(cArgumentTypes)-1), &cArgumentTypes[0])
	if cProbeDef == nil {
		return nil, fmt.Errorf("usdt_create_probe (%s:%s)", function, name)
	}

	return &Probe{
		Function:  function,
		Name:      name,
		cProbeDef: cProbeDef,
	}, nil
}

// Enabled returns true if the probe is enabled, false otherwise
func (p *Probe) Enabled() bool {
	if p.cProbeDef == nil {
		return false
	}

	cProbe := (*C.usdt_probe_t)(p.cProbeDef.probe)

	if C.int(C.usdt_is_enabled(cProbe)) == 1 {
		return true
	}

	return false
}

// Fire fires the probe with the supplied arguments
func (p *Probe) Fire(args ...interface{}) error {
	expectedArgC := p.cProbeDef.argc
	if C.size_t(len(args)) != expectedArgC {
		return fmt.Errorf("DTrace [%s:%s]: Expected %d arguments, got %d", p.Function, p.Name,
			expectedArgC, len(args))
	}

	cProbeArgs := make([]unsafe.Pointer, expectedArgC+1)
	for i, arg := range args {
		switch argT := arg.(type) {
		case int:
			val := C.int(argT)
			cProbeArgs[i] = unsafe.Pointer(uintptr(val))
		case string:
			val := C.CString(argT)
			defer C.free(unsafe.Pointer(val))
			cProbeArgs[i] = unsafe.Pointer(val)
		default:
			return fmt.Errorf("Arguments must be of kind Int or String, got %T", argT)
		}
	}

	cProbe := (*C.usdt_probe_t)(p.cProbeDef.probe)
	C.usdt_fire_probe(cProbe, C.size_t(len(cProbeArgs)-1), &cProbeArgs[0])

	return nil
}
