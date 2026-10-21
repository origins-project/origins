// What wasm_exec.js, copied from the Go that built chart.wasm, sets on the page.
declare global {
  class Go {
    importObject: WebAssembly.Imports
    run(instance: WebAssembly.Instance): Promise<void>
  }
}

export {}
