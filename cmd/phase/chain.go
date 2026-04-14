package phase

import (
	"fmt"

	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
)

// runCompileChain runs install → compile.
func runCompileChain(base string, c *commonFlags) error {
	if err := query.RunPackInstall(base, c.language); err != nil {
		return err
	}
	return query.RunCompile(base, c.language, "", c.numThreads)
}

// runTestChain runs install → compile → test.
func runTestChain(base string, c *commonFlags) error {
	if err := runCompileChain(base, c); err != nil {
		return err
	}
	return qlttest.RunUnitTests(base, c.language, c.codeqlArgs, c.numThreads)
}

// runVerifyChain runs install → compile → test → verify (placeholder).
func runVerifyChain(base string, c *commonFlags) error {
	if err := runTestChain(base, c); err != nil {
		return err
	}
	fmt.Println("verify: not yet fully implemented.")
	fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
	return nil
}
