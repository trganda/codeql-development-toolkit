package pack

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("SelectPacks", func() {
	It("returns the exact pack for a full name", func() {
		p := &Pack{
			YmlPath: filepath.Join("cpp", "acme", "qp", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "acme/qp"},
		}

		got, err := SelectPacks([]*Pack{p}, []string{"acme/qp"}, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0]).To(BeIdenticalTo(p))
	})

	It("returns the unique pack for a short name", func() {
		want := &Pack{
			YmlPath: filepath.Join("go", "mine", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "mine/myqueries"},
		}

		got, err := SelectPacks([]*Pack{want}, []string{"myqueries"}, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0]).To(BeIdenticalTo(want))
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

		_, err := SelectPacks([]*Pack{p1, p2}, []string{"foo"}, true)

		Expect(err).To(HaveOccurred())
		Expect(err.Error()).To(ContainSubstring("multiple"))
	})

	It("skips test packs when skipTest is true", func() {
		q := &Pack{
			YmlPath: filepath.Join("cpp", "pack", "src", "qlpack.yml"),
			Config:  QlpackConfig{Name: "scope/querypack"},
		}
		testPack := &Pack{
			YmlPath: filepath.Join("cpp", "pack", "test", "qlpack.yml"),
			Config:  QlpackConfig{Name: "scope/querypack-tests"},
		}

		got, err := SelectPacks([]*Pack{testPack, q}, []string{"scope/querypack"}, true)

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(1))
		Expect(got[0]).To(BeIdenticalTo(q))
	})

	It("returns all packs when names is empty", func() {
		p1 := &Pack{
			YmlPath: filepath.Join("a", "qlpack.yml"),
			Config:  QlpackConfig{Name: "x/a"},
		}
		p2 := &Pack{
			YmlPath: filepath.Join("b", "qlpack.yml"),
			Config:  QlpackConfig{Name: "y/b"},
		}

		got, err := SelectPacks([]*Pack{p1, p2}, nil, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(2))
	})

	It("resolves multiple names in one call", func() {
		p1 := &Pack{
			YmlPath: filepath.Join("a", "qlpack.yml"),
			Config:  QlpackConfig{Name: "x/a"},
		}
		p2 := &Pack{
			YmlPath: filepath.Join("b", "qlpack.yml"),
			Config:  QlpackConfig{Name: "y/b"},
		}

		got, err := SelectPacks([]*Pack{p1, p2}, []string{"x/a", "y/b"}, false)

		Expect(err).NotTo(HaveOccurred())
		Expect(got).To(HaveLen(2))
	})
})
