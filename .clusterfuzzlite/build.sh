#!/bin/bash -eu
# ClusterFuzzLite build script.
# Compiles each Go native fuzz target into an OSS-Fuzz / libFuzzer binary.

cd "$SRC/codeql-development-toolkit"

# Ensure module dependencies are available inside the build container.
go mod download

MODULE="github.com/trganda/codeql-development-toolkit"

compile_native_go_fuzzer "$MODULE/internal/archive" FuzzExtractTarGz fuzz_extract_targz
compile_native_go_fuzzer "$MODULE/internal/archive" FuzzExtractZip   fuzz_extract_zip
compile_native_go_fuzzer "$MODULE/internal/config"  FuzzLoadFromFile fuzz_load_config
compile_native_go_fuzzer "$MODULE/internal/bundle"  FuzzReadSarifResults fuzz_read_sarif
