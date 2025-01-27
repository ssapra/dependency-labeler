// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package integration_test

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/vmware-tanzu/dependency-labeler/pkg/metadata"

	"github.com/vmware-tanzu/dependency-labeler/test/test_utils"

	"github.com/onsi/gomega/ghttp"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deplab additional sources file", func() {
	Context("when I supply additional sources file(s) as an argument", func() {
		var (
			metadataLabel              metadata.Metadata
			additionalArguments        []string
			inputAdditionalSourcesPath string
			server                     *ghttp.Server
		)

		JustBeforeEach(func() {
			metadataLabel = runDeplabAgainstTar(getTestAssetPath("image-archives/tiny.tgz"), additionalArguments...)
		})

		Context("and it only has one source archive", func() {
			BeforeEach(func() {
				server = startServer(
					ghttp.RespondWith(http.StatusOK, []byte("HTTP status not found code returned")))

				inputAdditionalSourcesPath = templateAdditionalSource("sources-file-single-archive.yml", server.URL())
				additionalArguments = []string{"--additional-sources-file", inputAdditionalSourcesPath}
			})

			AfterEach(func() {
				test_utils.CleanupFile(inputAdditionalSourcesPath)
				server.Close()
			})

			It("adds a archive dependency", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(1))
				archiveDependency := archiveDependencies[0]
				Expect(archiveDependency.Source.Metadata).NotTo(BeNil())
				archiveSourceMetadata := archiveDependency.Source.Metadata.(map[string]interface{})
				Expect(archiveDependency.Type).ToNot(BeEmpty())

				By("adding the source archive url to the archive dependency")
				Expect(archiveDependency.Type).To(Equal("package"))
				Expect(archiveDependency.Source.Type).To(Equal(metadata.ArchiveType))
				Expect(archiveSourceMetadata["url"]).To(Equal(server.URL() + "/ubuntu/pool/main/c/ca-certificates/ca-certificates_20180409.tar.xz"))
			})
		})

		Context("with multiple source archive urls", func() {
			BeforeEach(func() {
				server = startServer(
					ghttp.RespondWith(http.StatusOK, ""),
					ghttp.RespondWith(http.StatusOK, ""),
				)

				inputAdditionalSourcesPath = templateAdditionalSource("sources-file-multiple-archives.yml", server.URL())

				additionalArguments = []string{"--additional-sources-file", inputAdditionalSourcesPath}
			})

			It("adds multiple archive url entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(2))
			})

			AfterEach(func() {
				test_utils.CleanupFile(inputAdditionalSourcesPath)
				server.Close()
			})
		})

		Context("with no source archive urls", func() {
			BeforeEach(func() {
				additionalArguments = []string{"--additional-sources-file", getTestAssetPath("sources/sources-file-empty-archives.yml")}
			})

			It("adds zero archive entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(0))
			})
		})

		Context("with only one vcs", func() {
			BeforeEach(func() {
				additionalArguments = []string{"--additional-sources-file", getTestAssetPath("sources/sources-file-single-vcs.yml")}
			})

			It("adds a git dependency", func() {
				gitDependencies := selectGitDependencies(metadataLabel.Dependencies)
				Expect(gitDependencies).To(HaveLen(2))
				vcsGitDependencies := selectVcsGitDependencies(gitDependencies)
				Expect(vcsGitDependencies).To(HaveLen(1))

				gitDependency := vcsGitDependencies[0]
				gitSourceMetadata := gitDependency.Source.Metadata.(map[string]interface{})
				Expect(gitDependency.Type).ToNot(BeEmpty())

				By("adding the git commit of HEAD to a git dependency")
				Expect(gitDependency.Type).To(Equal("package"))
				Expect(gitDependency.Source.Version["commit"]).To(Equal("abc123"))

				By("adding the git remote to a git dependency")
				Expect(gitSourceMetadata["url"].(string)).To(Equal("git@github.com:vmware-tanzu/dependency-labeler.git"))
			})
		})

		Context("with both multiple vcs and multiple archives", func() {
			BeforeEach(func() {
				server = startServer(
					ghttp.RespondWith(http.StatusOK, ""),
					ghttp.RespondWith(http.StatusOK, ""),
				)

				inputAdditionalSourcesPath = templateAdditionalSource("sources-file-multiple-archives-multiple-vcs.yml", server.URL())

				additionalArguments = []string{"--additional-sources-file", inputAdditionalSourcesPath}
			})

			It("adds git dependencies and archives", func() {
				gitDependencies := selectGitDependencies(metadataLabel.Dependencies)
				Expect(gitDependencies).To(HaveLen(3))
				vcsGitDependencies := selectVcsGitDependencies(gitDependencies)
				Expect(vcsGitDependencies).To(HaveLen(2))
			})

			AfterEach(func() {
				server.Close()
				test_utils.CleanupFile(inputAdditionalSourcesPath)
			})
		})

		Context("with multiple sources files", func() {
			var (
				inputAdditionalSourcesPath1 string
				inputAdditionalSourcesPath2 string
			)

			BeforeEach(func() {
				server = startServer(
					ghttp.RespondWith(http.StatusOK, ""),
					ghttp.RespondWith(http.StatusOK, ""),
					ghttp.RespondWith(http.StatusOK, ""),
				)

				inputAdditionalSourcesPath1 = templateAdditionalSource("sources-file-multiple-archives.yml", server.URL())
				inputAdditionalSourcesPath2 = templateAdditionalSource("sources-file-single-archive.yml", server.URL())

				additionalArguments = []string{
					"--additional-sources-file", inputAdditionalSourcesPath1,
					"--additional-sources-file", inputAdditionalSourcesPath2,
				}
			})

			It("adds multiple archiveDependency entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(3))
			})

			AfterEach(func() {
				server.Close()
				test_utils.CleanupFile(inputAdditionalSourcesPath1)
				test_utils.CleanupFile(inputAdditionalSourcesPath2)
			})
		})
	})

	Context("when I supply invalid additional sources file(s) as an argument", func() {

		Context("with erroneous paths", func() {
			It("exits with an error", func() {
				_, stdErr := runDepLab([]string{
					"--additional-sources-file", "erroneous_path.yml",
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "doesnotmatter1",
				}, 1)

				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("could not parse additional sources file: erroneous_path.yml"))
			})
		})

		Context("with an empty file", func() {
			It("exits with an error", func() {
				_, stdErr := runDepLab([]string{
					"--additional-sources-file", getTestAssetPath("sources/empty-file.yml"),
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "doesnotmatter2",
				}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("could not parse additional sources file"))
			})
		})

		Context("with a unsupported vcs type", func() {
			It("exits with an error", func() {
				_, stdErr := runDepLab([]string{
					"--additional-sources-file", getTestAssetPath("sources/sources-unsupported-vcs.yml"),
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "doesnotmatter3",
				}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("unsupported vcs protocol: hg"))
			})
		})

		Context("with a invalid file extension", func() {
			It("exits with an error", func() {
				_, stdErr := runDepLab([]string{
					"--additional-sources-file", getTestAssetPath("sources/sources-file-unsupported-extension.yml"),
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "doesnotmatter4",
				}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("unsupported extension for url"))
			})
		})

		Context("with an invalid git url", func() {
			It("exits with an error", func() {
				_, stdErr := runDepLab([]string{
					"--additional-sources-file", getTestAssetPath("sources/sources-invalid-git-url.yml"),
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "doesnotmatter5",
				}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(SatisfyAll(
					ContainSubstring("error"),
					ContainSubstring("vmware-tanzu/dependency-labeler.git"),
				))
			})

			Context("with ignore-validation-error flag set", func() {
				It("succeeds with a warning", func() {
					f, err := ioutil.TempFile("", "")
					Expect(err).ToNot(HaveOccurred())

					defer os.Remove(f.Name())

					_, stdErr := runDepLab([]string{
						"--additional-sources-file", getTestAssetPath("sources/sources-invalid-git-url.yml"),
						"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
						"--git", pathToGitRepo,
						"--metadata-file", f.Name(),
						"--ignore-validation-errors",
					}, 0)

					By("providing a warning message")
					errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
					Expect(errorOutput).To(SatisfyAll(
						ContainSubstring("warning"),
						ContainSubstring("vmware-tanzu/dependency-labeler.git"),
					))

					By("by including a vcs git dependency")
					metadataLabel := metadata.Metadata{}
					err = json.NewDecoder(f).Decode(&metadataLabel)
					Expect(err).ToNot(HaveOccurred())

					gitDependencies := selectGitDependencies(metadataLabel.Dependencies)
					Expect(gitDependencies).To(HaveLen(2))
					vcsGitDependencies := selectVcsGitDependencies(gitDependencies)
					Expect(vcsGitDependencies).To(HaveLen(1))
				})
			})
		})
	})
})

func selectVcsGitDependencies(dependencies []metadata.Dependency) []metadata.Dependency {
	var gitDependencies []metadata.Dependency
	for _, dependency := range dependencies {
		Expect(dependency.Source.Metadata).To(Not(BeNil()))
		gitSourceMetadata := dependency.Source.Metadata.(map[string]interface{})
		if dependency.Source.Type == metadata.GitSourceType && gitSourceMetadata["url"].(string) != "https://example.com/example.git" {
			gitDependencies = append(gitDependencies, dependency)
		}
	}
	return gitDependencies
}

func templateAdditionalSource(path, server string) string {
	inputAdditionalSourcesPath, err := filepath.Abs(filepath.Join("assets", "sources", path))
	Expect(err).ToNot(HaveOccurred())

	t, err := template.ParseFiles(inputAdditionalSourcesPath)
	Expect(err).ToNot(HaveOccurred())

	f, err := ioutil.TempFile("", "")
	Expect(err).ToNot(HaveOccurred())

	err = t.Execute(f, struct {
		Server string
	}{
		server,
	})
	Expect(err).ToNot(HaveOccurred())

	return f.Name()
}
