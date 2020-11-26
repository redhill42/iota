package mosquitto

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/redhill42/iota/auth"
	"github.com/redhill42/iota/auth/userdb"
	_ "github.com/redhill42/iota/auth/userdb/mongodb"
	"github.com/redhill42/iota/device"
)

func TestAuthPlugin(t *testing.T) {
	os.Setenv("IOTA_USERDB_URL", "mongodb://127.0.0.1:27017/mos_auth_test")
	os.Setenv("IOTA_DEVICEDB_URL", "mongodb://127.0.0.1:27017/mos_auth_test")

	RegisterFailHandler(Fail)
	RunSpecs(t, "AuthPlugin Suite")
}

var _ = Describe("AuthPlugin", func() {
	const (
		TEST_USER      = "test@auth-plugin.local"
		TEST_PASSWORD  = "test"
		TEST_DEVICE    = "test"
		OTHER_DEVICE   = "other"
		TEST_CLIENT_ID = "TEST_CLIENT"
	)

	var (
		db    *userdb.UserDatabase
		authz *auth.Authenticator
		mgr   *device.Manager

		userToken, deviceToken, otherToken string
	)

	BeforeEach(func() {
		var err error

		db, err = userdb.Open()
		Expect(err).NotTo(HaveOccurred())

		authz, err = auth.NewAuthenticator(db)
		Expect(err).NotTo(HaveOccurred())

		mgr, err = device.NewManager(nil)
		Expect(err).NotTo(HaveOccurred())

		user := userdb.BasicUser{Name: TEST_USER}
		Ω(db.Create(&user, TEST_PASSWORD)).Should(Succeed())
		_, userToken, err = authz.Authenticate(TEST_USER, TEST_PASSWORD)
		Expect(err).ShouldNot(HaveOccurred())

		deviceToken, err = mgr.CreateToken(TEST_DEVICE)
		Expect(err).NotTo(HaveOccurred())
		Ω(mgr.Create(TEST_DEVICE, deviceToken, device.Record{})).Should(Succeed())

		otherToken, err = mgr.CreateToken(OTHER_DEVICE)
		Expect(err).NotTo(HaveOccurred())
		Ω(mgr.Create(OTHER_DEVICE, otherToken, device.Record{})).Should(Succeed())

		Ω(AuthPluginInit(nil, nil, 0)).Should(BeTrue())
	})

	AfterEach(func() {
		db.Remove(TEST_USER)
		db.Close()

		mgr.Remove(TEST_DEVICE)
		mgr.Remove(OTHER_DEVICE)
		mgr.Close()

		AuthPluginCleanup()
	})

	Describe("Authentication", func() {
		Context("Authorized user", func() {
			It("should authenticate by mosquitto with user name and password", func() {
				Ω(AuthUnpwdCheck(TEST_USER, TEST_PASSWORD, TEST_CLIENT_ID)).Should(BeTrue())
			})

			It("should authenticate by mosquitto with access token", func() {
				Ω(AuthUnpwdCheck(userToken, "", TEST_CLIENT_ID)).Should(BeTrue())
			})
		})

		Context("Unauthorized user", func() {
			It("should reject by mosquitto", func() {
				Ω(AuthUnpwdCheck("nobody", "nobody", TEST_CLIENT_ID)).Should(BeFalse())
			})
		})

		Context("Authorized device", func() {
			It("should authenticate by mosquitto with access token", func() {
				Ω(AuthUnpwdCheck(deviceToken, "", TEST_CLIENT_ID)).Should(BeTrue())
			})
		})

		Context("Unauthorized device", func() {
			It("should reject by mosquitto", func() {
				Ω(AuthUnpwdCheck("FAKE_TOKEN", "", TEST_CLIENT_ID)).Should(BeFalse())
			})
		})

		Context("Anonymous device", func() {
			It("should accept by mosquitto for claiming", func() {
				Ω(AuthUnpwdCheck("", "", TEST_CLIENT_ID)).Should(BeTrue())
			})
		})
	})

	Describe("Authorization", func() {
		Context("Authorized user", func() {
			It("should have access to any topic", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, TEST_USER, "test/topic", _MOSQ_ACL_READ)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, TEST_USER, "test/topic", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, TEST_USER, "test/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeTrue())

				Ω(AuthAclCheck(TEST_CLIENT_ID, userToken, "test/topic", _MOSQ_ACL_READ)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, userToken, "test/topic", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, userToken, "test/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeTrue())
			})
		})

		Context("Authorized device", func() {
			It("can write to api request topic", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/"+deviceToken+"/me/attributes", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/"+otherToken+"/me/attributes", _MOSQ_ACL_WRITE)).Should(BeTrue())
			})

			It("cannot read from api request topic", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/"+deviceToken+"/me/attributes", _MOSQ_ACL_READ)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/"+otherToken+"/me/attributes", _MOSQ_ACL_READ)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/+/me/attributes", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
			})

			It("can read/write api response topics for itself or other devices", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, deviceToken+"/me/attributes", _MOSQ_ACL_READ)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, deviceToken+"/me/attributes", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, deviceToken+"/me/attributes/response/+", _MOSQ_ACL_SUBSCRIBE)).Should(BeTrue())

				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, otherToken+"/me/attributes", _MOSQ_ACL_READ)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, otherToken+"/me/attributes", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, otherToken+"/me/attributes/response/+", _MOSQ_ACL_SUBSCRIBE)).Should(BeTrue())
			})

			It("should have full access for non-api topics", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "test/topic", _MOSQ_ACL_READ)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "test/topic", _MOSQ_ACL_WRITE)).Should(BeTrue())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "test/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeTrue())
			})

			It("should not subscribe on wildcard topic that cover api topic", func() {
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "api/v1/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "+/v1/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "+/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "+/+/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "+/+/+/#", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
				Ω(AuthAclCheck(TEST_CLIENT_ID, deviceToken, "+/+/attributes", _MOSQ_ACL_SUBSCRIBE)).Should(BeFalse())
			})
		})
	})
})
