package bundle

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("platformExclusions", func() {
	It("excludes osx and win tools subdirs for linux64 and keeps linux subdirs", func() {
		excl := platformExclusions("linux64", []string{"cpp"})

		Expect(excl).To(ContainElement("tools/osx64"))
		Expect(excl).To(ContainElement("tools/macos"))
		Expect(excl).To(ContainElement("tools/win64"))
		Expect(excl).To(ContainElement("tools/windows"))
		Expect(excl).To(ContainElement("cpp/tools/osx64"))
		Expect(excl).To(ContainElement("cpp/tools/win64"))

		Expect(excl).NotTo(ContainElement("tools/linux64"))
		Expect(excl).NotTo(ContainElement("tools/linux"))
		Expect(excl).NotTo(ContainElement("cpp/tools/linux64"))
	})

	It("excludes codeql.exe and swift osx64 dirs for linux64", func() {
		excl := platformExclusions("linux64", nil)

		Expect(excl).To(ContainElement("codeql.exe"))
		Expect(excl).To(ContainElement("swift/qltest/osx64"))
		Expect(excl).To(ContainElement("swift/resource-dir/osx64"))
	})

	It("excludes linux and win tools subdirs for osx64 and adds swift linux64 paths", func() {
		excl := platformExclusions("osx64", []string{"java"})

		Expect(excl).To(ContainElement("tools/linux64"))
		Expect(excl).To(ContainElement("tools/win64"))
		Expect(excl).To(ContainElement("java/tools/linux64"))

		Expect(excl).NotTo(ContainElement("tools/osx64"))
		Expect(excl).NotTo(ContainElement("tools/macos"))

		Expect(excl).To(ContainElement("codeql.exe"))
		Expect(excl).To(ContainElement("swift/qltest/linux64"))
		Expect(excl).To(ContainElement("swift/resource-dir/linux64"))
	})

	It("excludes linux and osx tools subdirs for win64 and keeps codeql.exe", func() {
		excl := platformExclusions("win64", []string{"go"})

		Expect(excl).To(ContainElement("tools/linux64"))
		Expect(excl).To(ContainElement("tools/osx64"))
		Expect(excl).To(ContainElement("go/tools/macos"))

		Expect(excl).NotTo(ContainElement("codeql.exe"))
		Expect(excl).NotTo(ContainElement("tools/win64"))
		Expect(excl).NotTo(ContainElement("tools/windows"))

		Expect(excl).To(ContainElement("swift/qltest"))
		Expect(excl).To(ContainElement("swift/resource-dir"))
	})

	It("expands per-language tools paths for every language passed in", func() {
		excl := platformExclusions("linux64", []string{"cpp", "java", "go"})

		for _, lang := range []string{"cpp", "java", "go"} {
			Expect(excl).To(ContainElement(lang + "/tools/osx64"))
			Expect(excl).To(ContainElement(lang + "/tools/win64"))
		}
	})
})

var _ = Describe("makePlatformFilter", func() {
	It("rejects an exact-match exclusion", func() {
		filter := makePlatformFilter("linux64", []string{"cpp"})

		Expect(filter("tools/osx64")).To(BeFalse())
		Expect(filter("codeql.exe")).To(BeFalse())
	})

	It("rejects paths under an excluded prefix", func() {
		filter := makePlatformFilter("linux64", []string{"cpp"})

		Expect(filter("tools/osx64/some/file")).To(BeFalse())
		Expect(filter("cpp/tools/win64/sub/file")).To(BeFalse())
	})

	It("admits paths that share a name prefix but are not under the excluded directory", func() {
		filter := makePlatformFilter("linux64", nil)

		Expect(filter("tools/osx64-extra")).To(BeTrue())
		Expect(filter("tools/osx64.txt")).To(BeTrue())
	})

	It("admits paths belonging to the target platform", func() {
		filter := makePlatformFilter("linux64", []string{"cpp"})

		Expect(filter("tools/linux64/runner")).To(BeTrue())
		Expect(filter("cpp/tools/linux/file")).To(BeTrue())
		Expect(filter("codeql")).To(BeTrue())
	})
})
