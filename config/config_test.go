package config_test

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "github.com/cloudfoundry/mkman/Godeps/_workspace/src/github.com/onsi/ginkgo"
	. "github.com/cloudfoundry/mkman/Godeps/_workspace/src/github.com/onsi/gomega"
	"github.com/cloudfoundry/mkman/config"
)

var _ = Describe("Config", func() {
	var (
		c config.Config

		tempDir string
	)

	BeforeEach(func() {
		var err error
		tempDir, err = ioutil.TempDir("", "")
		Expect(err).NotTo(HaveOccurred())

		cfPath := filepath.Join(tempDir, "cf")
		err = os.MkdirAll(cfPath, os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		stemcellPath := filepath.Join(tempDir, "stemcell.tgz")
		err = ioutil.WriteFile(stemcellPath, []byte("some content"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		etcdPath := filepath.Join(tempDir, "etcd.tgz")
		err = ioutil.WriteFile(etcdPath, []byte("some content"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		consulPath := filepath.Join(tempDir, "consul.tgz")
		err = ioutil.WriteFile(consulPath, []byte("some content"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		stubPath0 := filepath.Join(tempDir, "stub0.yml")
		err = ioutil.WriteFile(stubPath0, []byte("---"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		stubPath1 := filepath.Join(tempDir, "stub1.yml")
		err = ioutil.WriteFile(stubPath1, []byte("---"), os.ModePerm)
		Expect(err).NotTo(HaveOccurred())

		c = config.Config{
			CFPath:       cfPath,
			StemcellPath: stemcellPath,
			EtcdPath:     etcdPath,
			ConsulPath:   consulPath,
			StubPaths:    []string{stubPath0, stubPath1},
		}
	})

	AfterEach(func() {
		err := os.RemoveAll(tempDir)
		Expect(err).NotTo(HaveOccurred())
	})

	It("should not return any error", func() {
		err := c.Validate()
		Expect(err).NotTo(HaveOccurred())
	})

	Context("with etcd", func() {
		Context("when the path is, in fact, a directory", func() {
			BeforeEach(func() {
				etcdPath := filepath.Join(tempDir, "etcd-as-a-dir")
				err := os.MkdirAll(etcdPath, os.ModePerm)
				Expect(err).NotTo(HaveOccurred())

				c.EtcdPath = etcdPath
			})

			It("should not return an error", func() {
				err := c.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})

		Context("when the path is set to director-latest", func() {
			BeforeEach(func() {
				c.EtcdPath = "director-latest"
			})

			It("should not return an error", func() {
				err := c.Validate()
				Expect(err).NotTo(HaveOccurred())
			})
		})
	})

	Context("when the consul path is, in fact, a directory", func() {
		BeforeEach(func() {
			consulPath := filepath.Join(tempDir, "consul-as-a-dir")
			err := os.MkdirAll(consulPath, os.ModePerm)
			Expect(err).NotTo(HaveOccurred())

			c.ConsulPath = consulPath
		})

		It("should not return an error", func() {
			err := c.Validate()
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Handling errors", func() {
		Describe("Error Headers", func() {
			BeforeEach(func() {
				c.CFPath = ""
				c.StemcellPath = ""
				c.EtcdPath = ""
				c.ConsulPath = ""
				c.StubPaths = []string{""}
			})

			It("Displays the fields the errors happened on", func() {
				err := c.Validate()
				Expect(err.Error()).To(ContainSubstring("there were 5 errors with 'config':"))
				Expect(err.Error()).To(ContainSubstring("there was 1 error with 'cf':"))
				Expect(err.Error()).To(ContainSubstring("there was 1 error with 'stemcell':"))
				Expect(err.Error()).To(ContainSubstring("there was 1 error with 'etcd':"))
				Expect(err.Error()).To(ContainSubstring("there was 1 error with 'consul':"))
				Expect(err.Error()).To(ContainSubstring("there was 1 error with 'stubs':"))
			})
		})

		Describe("on the CFPath", func() {
			Context("when it is an empty string", func() {
				BeforeEach(func() {
					c.CFPath = ""
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))
				})
			})

			Context("when it is not an absolute path", func() {
				BeforeEach(func() {
					c.CFPath = "./path/to/cf"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be absolute path"))
					Expect(err.Error()).To(ContainSubstring(c.CFPath))
				})
			})

			Context("when the directory does not exist", func() {
				BeforeEach(func() {
					c.CFPath = "/path/to/invalid/directory"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("cf must be a path to a directory that exists: '%s'", c.CFPath))
				})
			})

			Context("when the directory is, in fact, a file", func() {
				BeforeEach(func() {
					cfPath := filepath.Join(tempDir, "cf-path-not-dir")
					err := ioutil.WriteFile(cfPath, []byte("some contents"), os.ModePerm)
					Expect(err).NotTo(HaveOccurred())

					c.CFPath = cfPath
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be a directory: %s", c.CFPath))
					Expect(err.Error()).To(ContainSubstring(c.CFPath))
				})
			})
		})

		Describe("on the StemcellPath", func() {
			Context("when it is an empty string", func() {
				BeforeEach(func() {
					c.StemcellPath = ""
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))
				})
			})

			Context("when it is not an absolute path", func() {
				BeforeEach(func() {
					c.StemcellPath = "./path/to/stemcell"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be absolute path"))
					Expect(err.Error()).To(ContainSubstring(c.StemcellPath))
				})
			})

			Context("when the stemcell file does not exist", func() {
				BeforeEach(func() {
					c.StemcellPath = "/path/to/invalid/stemcell"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("stemcell must be a path to a file that exists: '%s'", c.StemcellPath))
				})
			})

			Context("when the stemcell path is, in fact, a directory", func() {
				BeforeEach(func() {
					stemcellPath := filepath.Join(tempDir, "stemcell-as-a-dir")
					err := os.MkdirAll(stemcellPath, os.ModePerm)
					Expect(err).NotTo(HaveOccurred())

					c.StemcellPath = stemcellPath
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be a file: %s", c.StemcellPath))
					Expect(err.Error()).To(ContainSubstring(c.StemcellPath))
				})
			})
		})

		Describe("on the EtcdPath", func() {
			Context("when it is an empty string", func() {
				BeforeEach(func() {
					c.EtcdPath = ""
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))
				})
			})

			Context("when it is not an absolute path", func() {
				BeforeEach(func() {
					c.EtcdPath = "./path/to/etcd"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp(".*must be valid version alias or absolute path: %s", c.EtcdPath))
					Expect(err.Error()).To(ContainSubstring(c.EtcdPath))
				})
			})

			Context("when the etcd file does not exist", func() {
				BeforeEach(func() {
					c.EtcdPath = "/path/to/invalid/etcd"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("value must be valid version alias or a path to a file or a directory that exists: %s", c.EtcdPath))
				})
			})
		})

		Describe("on the ConsulPath", func() {
			Context("when it is an empty string", func() {
				BeforeEach(func() {
					c.ConsulPath = ""
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))
				})
			})

			Context("when it is not an absolute path", func() {
				BeforeEach(func() {
					c.ConsulPath = "./path/to/consul"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp(".*must be valid version alias or absolute path: %s", c.ConsulPath))
					Expect(err.Error()).To(ContainSubstring(c.ConsulPath))
				})
			})

			Context("when the consul file does not exist", func() {
				BeforeEach(func() {
					c.ConsulPath = "/path/to/invalid/consul"
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("value must be valid version alias or a path to a file or a directory that exists: %s", c.ConsulPath))
					Expect(err.Error()).To(ContainSubstring(c.ConsulPath))
				})
			})
		})
		Describe("on the StubPaths", func() {
			Context("when there are no stub paths", func() {
				BeforeEach(func() {
					c.StubPaths = []string{}
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty array: stubs"))
				})
			})

			Context("when there is an empty stub path", func() {
				BeforeEach(func() {
					c.StubPaths = []string{""}
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))
				})
			})

			Context("when it is not an absolute path", func() {
				BeforeEach(func() {
					c.StubPaths = []string{"./path/to/stub"}
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(ContainSubstring("value must be absolute path"))
					Expect(err.Error()).To(ContainSubstring(c.StubPaths[0]))
				})
			})

			Context("when the stub file does not exist", func() {
				BeforeEach(func() {
					c.StubPaths = []string{"/path/to/invalid/stub"}
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("must be a path to a file that exists: '%s'", c.StubPaths[0]))
				})
			})

			Context("when the stub path is, in fact, a directory", func() {
				BeforeEach(func() {
					stubPath0 := filepath.Join(tempDir, "stub-as-a-dir")
					err := os.MkdirAll(stubPath0, os.ModePerm)
					Expect(err).NotTo(HaveOccurred())

					c.StubPaths = []string{stubPath0}
				})

				It("should return an error", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())
					Expect(err.Error()).To(MatchRegexp("value must be a file.*%s", c.StubPaths[0]))
				})
			})
		})

		Describe("multiple errors", func() {
			BeforeEach(func() {
				c.CFPath = ""
				c.StubPaths = []string{"/not/a/valid/stub", "/also/not/a/valid/stub"}
			})

			Context("when there are multiple errors", func() {
				It("should return the errors", func() {
					err := c.Validate()
					Expect(err).To(HaveOccurred())

					Expect(err.Error()).To(ContainSubstring("value must be non-empty"))

					Expect(err.Error()).To(ContainSubstring("must be a path to a file that exists:"))
					Expect(err.Error()).To(ContainSubstring(c.StubPaths[0]))

					Expect(err.Error()).To(ContainSubstring("must be a path to a file that exists:"))
					Expect(err.Error()).To(ContainSubstring(c.StubPaths[1]))
				})
			})
		})
	})
})
