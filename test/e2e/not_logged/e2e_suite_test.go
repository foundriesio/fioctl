package not_logged

import (
	//nolint:golint
	//nolint:revive
	. "github.com/onsi/ginkgo/v2"

	//nolint:golint
	//nolint:revive
	. "github.com/onsi/gomega"

	"github.com/foundriesio/fioctl/test/e2e/utils"
)

var _ = Describe("Fioctl CLI", func() {
	Context("without login", func() {
		var tc *utils.TestContext

		BeforeEach(func() {
			var err error
			By("creating context")
			tc, err = utils.NewTestContext(utils.FioCLIBinName)
			Expect(err).NotTo(HaveOccurred())

		})

		It("should successfully show the help", func() {
			_, err := tc.Ctl("help")
			Expect(err).To(BeNil())
		})

		It("should successfully logout", func() {
			_, err := tc.Ctl("logout")
			Expect(err).To(BeNil())
		})

		It("should fail by trying get data without login", func() {
			output, err := tc.Ctl("targets", "list")
			Expect(err).To(HaveOccurred(), output)
			Expect(output).To(ContainSubstring("ERROR: Please run: \"fioctl login\" first"))
		})
	})
})
