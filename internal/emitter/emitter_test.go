package emitter_test

import (
	. "github.com/newrelic/nri-network-telemetry/internal/emitter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Emitter", func() {
	var (
		conf EmitConfig
	)

	Describe("Instantiated Emitter", func() {
		Context("configured to use STDOUT", func() {
			em := New("LOG", conf, nil)
			It("Should create a new emitter", func() {
				Expect(em).To(Equal("STDOUT"))
			})
		})

	})

})
