package pack

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SelectPackForAnalyze", func() {
	It("returns the exact pack for a full name", func() {
		p := &Pack{
			YmlPath: filepath.Join("cpp", "acme", "qp", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "acme/qp"},
		}

		got, err := SelectPackForAnalyze([]*Pack{p}, "acme/qp")

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeIdenticalTo(p))
	})

	It("returns the unique pack for a short name", func() {
		want := &Pack{
			YmlPath: filepath.Join("go", "mine", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "mine/myqueries"},
		}

		got, err := SelectPackForAnalyze([]*Pack{want}, "myqueries")

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeIdenticalTo(want))
	})

	It("returns an ambiguity error for duplicate short names", func() {
		p1 := &Pack{
			YmlPath: filepath.Join("java", "a", "foo", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "x/foo"},
		}
		p2 := &Pack{
			YmlPath: filepath.Join("java", "b", "foo", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "y/foo"},
		}

		_, err := SelectPackForAnalyze([]*Pack{p1, p2}, "foo")

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple"))
	})

	It("skips test packs when resolving the query pack", func() {
		q := &Pack{
			YmlPath: filepath.Join("cpp", "pack", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "scope/querypack"},
		}
		testPack := &Pack{
			YmlPath: filepath.Join("cpp", "pack", "test", "qlpack.yml"),
			Config:  QlpackConfig{Name: "scope/querypack-tests"},
		}

		got, err := SelectPackForAnalyze([]*Pack{testPack, q}, "scope/querypack")

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(BeIdenticalTo(q))
	})
})
