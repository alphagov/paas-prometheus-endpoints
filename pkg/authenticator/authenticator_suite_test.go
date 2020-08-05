package authenticator_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestAuthenticator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Authenticator Suite")
}
