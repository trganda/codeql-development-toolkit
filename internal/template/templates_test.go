package template

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func renderTmpl(path string, data any) string {
	GinkgoHelper()
	src, err := Get(path)
	Expect(err).NotTo(HaveOccurred())
	out, err := Render(src, data)
	Expect(err).NotTo(HaveOccurred())
	return out
}

var _ = Describe("install-qlt template", func() {
	const path = "shared/actions/install-qlt.tmpl"

	It("renders without any template data", func() {
		out := renderTmpl(path, nil)
		Expect(out).To(ContainSubstring("name: Fetch and Install QLT"))
		Expect(out).To(ContainSubstring("actions/setup-go@v6"))
		Expect(out).To(ContainSubstring("go install github.com/trganda/codeql-development-toolkit/cmd/qlt"))
	})

	It("contains no unrendered template directives", func() {
		out := renderTmpl(path, nil)
		Expect(out).NotTo(ContainSubstring("[["))
		Expect(out).NotTo(ContainSubstring("]]"))
	})
})

var _ = Describe("run-unit-tests template", func() {
	const path = "test/actions/run-unit-tests.tmpl"

	baseOpts := TestInitOptions{
		Language:   "cpp",
		LangFlag:   "cpp",
		Branch:     "main",
		NumThreads: 4,
		UseRunner:  "ubuntu-latest",
	}

	It("substitutes the workflow title and branch", func() {
		out := renderTmpl(path, baseOpts)
		Expect(out).To(ContainSubstring("name: ⚙️ CodeQL - Run Unit Tests (cpp)"))
		Expect(out).To(ContainSubstring("- 'main'"))
		Expect(out).To(ContainSubstring("--os-version ubuntu-latest"))
	})

	It("renders --language when LangFlag is set", func() {
		out := renderTmpl(path, baseOpts)
		Expect(out).To(ContainSubstring("qlt phase test --language cpp"))
	})

	It("omits --language when LangFlag is empty (testing all languages)", func() {
		opts := baseOpts
		opts.LangFlag = ""
		out := renderTmpl(path, opts)
		Expect(out).To(ContainSubstring("qlt phase test\n"))
		Expect(out).NotTo(ContainSubstring("--language"))
	})

	It("omits --codeql-args when CodeqlArgs is empty", func() {
		out := renderTmpl(path, baseOpts)
		Expect(out).NotTo(ContainSubstring("--codeql-args"))
	})

	It("renders --codeql-args when CodeqlArgs is non-empty", func() {
		opts := baseOpts
		opts.CodeqlArgs = "--ram=8000"
		out := renderTmpl(path, opts)
		Expect(out).To(ContainSubstring(`--codeql-args "--ram=8000"`))
	})

	It("includes --num-threads from NumThreads", func() {
		opts := baseOpts
		opts.NumThreads = 8
		out := renderTmpl(path, opts)
		Expect(out).To(ContainSubstring("--num-threads 8"))
	})

	It("contains no unrendered template directives", func() {
		out := renderTmpl(path, baseOpts)
		Expect(out).NotTo(ContainSubstring("[["))
		Expect(out).NotTo(ContainSubstring("]]"))
	})
})

var _ = Describe("run-bundle-integration-tests template", func() {
	const path = "bundle/actions/run-bundle-integration-tests.tmpl"

	It("renders the matrix with a single language", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp"}, Branch: "main",
		})
		Expect(out).To(ContainSubstring("name: ⚙️ Integration Test Bundle (cpp)"))
		Expect(out).To(ContainSubstring("language: [ 'cpp' ]"))
	})

	It("renders the matrix with multiple languages comma-separated", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp", "java"}, Branch: "main",
		})
		Expect(out).To(ContainSubstring("name: ⚙️ Integration Test Bundle (cpp, java)"))
		Expect(out).To(ContainSubstring("language: [ 'cpp', 'java' ]"))
	})

	It("substitutes the branch into push and pull_request triggers", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp"}, Branch: "release",
		})
		Expect(out).To(ContainSubstring("- 'release'"))
	})

	It("omits the config block when no packs are bundled", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp"}, Branch: "main",
		})
		Expect(out).NotTo(ContainSubstring("config: |"))
		Expect(out).NotTo(ContainSubstring("disable-default-queries"))
		Expect(out).NotTo(ContainSubstring("packs:"))
	})

	It("emits a config block with disable-default-queries and the packs list", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp"}, Branch: "main",
			Packs: []string{"qlt55/stuff", "team/another-pack"},
		})
		Expect(out).To(ContainSubstring("config: |"))
		Expect(out).To(ContainSubstring("disable-default-queries: true"))
		Expect(out).To(ContainSubstring("packs:\n            - qlt55/stuff\n            - team/another-pack"))
	})

	It("contains no unrendered template directives", func() {
		out := renderTmpl(path, BundleInitOptions{
			Languages: []string{"cpp"}, Branch: "main",
			Packs: []string{"qlt55/stuff"},
		})
		Expect(out).NotTo(ContainSubstring("[["))
		Expect(out).NotTo(ContainSubstring("]]"))
	})
})
