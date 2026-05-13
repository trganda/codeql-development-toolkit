#!/bin/bash -eu
# ClusterFuzzLite build script.
# Compiles each Go native fuzz target into an OSS-Fuzz / libFuzzer binary.

cd "$SRC/codeql-development-toolkit"

# compile_native_go_fuzzer rewrites testing.F calls to use the helper package
# below, so it must be in the module graph during the fuzzer build. This only
# affects the ephemeral build container; the repo's go.mod is unchanged.
# Do NOT run `go mod tidy` afterwards — it would strip the dep back out
# because no source file in the repo imports it directly.
go get github.com/AdamKorcz/go-118-fuzz-build/testing

MODULE="github.com/trganda/codeql-development-toolkit"

compile_native_go_fuzzer "$MODULE/internal/archive" FuzzExtractTarGz fuzz_extract_targz
compile_native_go_fuzzer "$MODULE/internal/archive" FuzzExtractZip   fuzz_extract_zip
compile_native_go_fuzzer "$MODULE/internal/config"  FuzzLoadFromFile fuzz_load_config
compile_native_go_fuzzer "$MODULE/internal/bundle"  FuzzReadSarifResults fuzz_read_sarif
