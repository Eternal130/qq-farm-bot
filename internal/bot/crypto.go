package bot

import (
	"context"
	_ "embed"
	"fmt"
	"sync"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
)

//go:embed tsdk.wasm
var tsdkWasm []byte

// Crypto handles message body encryption using the tsdk WASM module.
// The game server requires outgoing protobuf message bodies to be encrypted
// before wrapping in the GateMessage envelope.
// Only outgoing bodies need encryption; server responses are not encrypted.
type Crypto struct {
	mu        sync.Mutex
	runtime   wazero.Runtime
	mod       api.Module
	fnEncrypt api.Function // J(ptr, len) - encrypt buffer in-place
	fnAlloc   api.Function // z(len) -> ptr - allocate WASM memory
	fnFree    api.Function // A(ptr) - free WASM memory
}

// NewCrypto initializes the WASM-based encryption module.
// The WASM module (tsdk.wasm) is embedded at compile time and loaded via wazero.
func NewCrypto() (*Crypto, error) {
	ctx := context.Background()
	r := wazero.NewRuntime(ctx)

	// Compile module first to inspect import signatures
	compiled, err := r.CompileModule(ctx, tsdkWasm)
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("compile wasm: %w", err)
	}

	// The WASM module imports a host module "a" with 21 no-op stub functions.
	// We dynamically match their signatures from the compiled module.
	builder := r.NewHostModuleBuilder("a")
	for _, fn := range compiled.ImportedFunctions() {
		moduleName, name, ok := fn.Import()
		if !ok || moduleName != "a" {
			continue
		}
		paramTypes := fn.ParamTypes()
		resultTypes := fn.ResultTypes()
		nResults := len(resultTypes)
		builder.NewFunctionBuilder().
			WithGoModuleFunction(
				api.GoModuleFunc(func(_ context.Context, _ api.Module, stack []uint64) {
					// No-op stub: zero out result positions
					for i := 0; i < nResults; i++ {
						stack[i] = 0
					}
				}),
				paramTypes,
				resultTypes,
			).
			Export(name)
	}
	if _, err := builder.Instantiate(ctx); err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("host module: %w", err)
	}

	// Instantiate the WASM module
	mod, err := r.InstantiateModule(ctx, compiled, wazero.NewModuleConfig())
	if err != nil {
		r.Close(ctx)
		return nil, fmt.Errorf("instantiate wasm: %w", err)
	}

	// Call runtime init — matches Node.js: try { exports.E(); } catch {}
	if initFn := mod.ExportedFunction("E"); initFn != nil {
		initFn.Call(ctx) // ignore error
	}

	c := &Crypto{
		runtime:   r,
		mod:       mod,
		fnEncrypt: mod.ExportedFunction("J"),
		fnAlloc:   mod.ExportedFunction("z"),
		fnFree:    mod.ExportedFunction("A"),
	}

	if c.fnEncrypt == nil || c.fnAlloc == nil || c.fnFree == nil {
		r.Close(ctx)
		return nil, fmt.Errorf("missing required WASM exports (J, z, A)")
	}

	return c, nil
}

// EncryptBody encrypts a protobuf message body for sending to the game server.
// Returns the encrypted body. Thread-safe.
func (c *Crypto) EncryptBody(body []byte) ([]byte, error) {
	if len(body) == 0 {
		return body, nil
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	ctx := context.Background()

	// Allocate buffer in WASM memory
	results, err := c.fnAlloc.Call(ctx, uint64(len(body)))
	if err != nil {
		return nil, fmt.Errorf("wasm alloc: %w", err)
	}
	ptr := uint32(results[0])

	// Copy input data to WASM memory
	if !c.mod.Memory().Write(ptr, body) {
		c.fnFree.Call(ctx, uint64(ptr))
		return nil, fmt.Errorf("wasm memory write failed (ptr=%d, len=%d)", ptr, len(body))
	}

	// Encrypt in-place
	if _, err := c.fnEncrypt.Call(ctx, uint64(ptr), uint64(len(body))); err != nil {
		c.fnFree.Call(ctx, uint64(ptr))
		return nil, fmt.Errorf("wasm encrypt: %w", err)
	}

	// Read encrypted data back
	encrypted, ok := c.mod.Memory().Read(ptr, uint32(len(body)))
	if !ok {
		c.fnFree.Call(ctx, uint64(ptr))
		return nil, fmt.Errorf("wasm memory read failed (ptr=%d, len=%d)", ptr, len(body))
	}

	// Copy to own slice (WASM memory buffer may be reused)
	result := make([]byte, len(encrypted))
	copy(result, encrypted)

	// Free WASM buffer
	c.fnFree.Call(ctx, uint64(ptr))

	return result, nil
}

// Close releases all WASM runtime resources.
func (c *Crypto) Close() error {
	return c.runtime.Close(context.Background())
}
