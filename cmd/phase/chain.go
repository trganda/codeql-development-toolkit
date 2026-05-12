package phase

import (
	"fmt"

	"github.com/trganda/codeql-development-toolkit/internal/query"
	qlttest "github.com/trganda/codeql-development-toolkit/internal/test"
	"github.com/trganda/codeql-development-toolkit/internal/utils"
)

// runCompileChain runs install → compile.
func runCompileChain(base string, c *utils.CommonFlags) error {
	// if err := query.RunPackInstall(base, c.language); err != nil {
	// 	return err
	// }
	return query.RunCompile(base, c)
}

// runTestChain runs install → compile → test. output controls whether
// a test report is written: nil means no report, a pointer to "" resolves to
// RunUnitTests's default <base>/target/test/ path, any other value is used as-is.
func runTestChain(base string, output string, c *utils.CommonFlags) error {
	if err := runCompileChain(base, c); err != nil {
		return err
	}
	return qlttest.RunUnitTests(base, c, output)
}

// runVerifyChain runs install → compile → test → verify (placeholder).
func runVerifyChain(base string, c *utils.CommonFlags) error {
	if err := runTestChain(base, "", c); err != nil {
		return err
	}
	fmt.Println("verify: not yet fully implemented.")
	fmt.Println("Run 'qlt validation run check-queries --language <lang>' for available checks.")
	return nil
}
