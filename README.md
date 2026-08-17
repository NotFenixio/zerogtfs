# ZeroGTFS
A binary format for GTFS, achieving a lot of compression.

# Benchmarks
Using highly advanced benchmarking tools (my laptop and a random GTFS source), we confirm that ZeroGTFS can achieve "up to" x40 compression.

# How to use it
This is designed to be used as a shared lib, so you can use it with FFI with your language of choice.

1. Head over to https://github.com/NotFenixio/zerogtfs/releases/latest
2. Select your system's file:
  - macOS: libzerogtfs.dylib
  - Windows: libzerogtfs.dll
  - Linux/Unix-like: libzerogtfs.so
  - To make C shut up about warnings, you can grab the libzerogtfs.h file too, and include it in the same level as your libzerogtfs.[ext]
3. Load the GenerateZeroGTFS function using FFI It takes two parameters: origin, which can be a path or URL, and output, where the .zgts file will be saved.
Example using Deno:
```ts
const ext = Deno.build.os === "windows" ? "dll" : Deno.build.os === "darwin" ? "dylib" : "so";
const libPath = `./libzerogtfs.${ext}`;

// Load the go lib via FFI
const dylib = Deno.dlopen(libPath, {
    GenerateZeroGTFS: {
        parameters: ["buffer", "buffer"], // CStrings: input (origin) and output (.zgts)
        result: "i32",
    },
});

function toCString(str: string): Uint8Array {
    return new TextEncoder().encode(str + "\0");
}

export const generateZeroGTFS = (source: string, output: string) => {
        // run our awesome lib function
        const success = dylib.symbols.GenerateZeroGTFS(
            toCString(source),
            toCString(output)
        );

        if (success !== 1) {
            console.log("Could not generate zeroGTFS file: {error}", { success })
            return false
        }
        return true
}
```
