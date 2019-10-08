package integration_test

import (
	"context"
	"github.com/docker/docker/api/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pivotal/deplab/metadata"
	"github.com/pivotal/deplab/providers"
	"path/filepath"
	"strings"
)

var _ = Describe("deplab additional sources file", func(){

	Context("when I supply an additional sources file as an argument", func() {
		var (
			metadataLabel       metadata.Metadata
			additionalArguments []string
			outputImage         string
		)

		JustBeforeEach(func() {
			inputImage := "ubuntu:bionic"
			outputImage, _, metadataLabel, _ = runDeplabAgainstImage(inputImage, additionalArguments...)
		})

		AfterEach(func() {
			_, err := dockerCli.ImageRemove(context.TODO(), outputImage, types.ImageRemoveOptions{})
			Expect(err).ToNot(HaveOccurred())
		})

		Context("when I supply an additional sources file with only one source archive", func() {
			BeforeEach(func() {
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "sources-file-single-archive.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath}
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
				Expect(archiveDependency.Source.Type).To(Equal(providers.ArchiveType))
				Expect(archiveSourceMetadata["url"]).To(Equal("http://archive.ubuntu.com/ubuntu/pool/main/c/ca-certificates/ca-certificates_20180409.tar.xz"))
			})
		})

		Context("when I supply an additional sources file with multiple source archive urls", func() {
			BeforeEach(func() {
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "sources-file-multiple-archives.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath}
			})

			It("adds multiple archive url entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(2))
			})
		})

		Context("when I supply an additional sources file with no source archive urls", func() {
			BeforeEach(func() {
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "sources-file-empty-archives.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath}
			})

			It("adds zero archive entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(0))
			})
		})

		Context("when I supply an additional sources file with only one vcs", func() {
			BeforeEach(func() {
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "sources-file-single-vcs.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath}
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
				Expect(gitSourceMetadata["url"].(string)).To(Equal("git@github.com:pivotal/deplab.git"))
			})
		})

		Context("when I supply an artefacts file with both multiple vcs and multiple archives", func() {
			BeforeEach(func() {
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "sources-file-multiple-archives-multiple-vcs.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath}
			})

			It("adds git dependencies and archives", func(){
				gitDependencies := selectGitDependencies(metadataLabel.Dependencies)
				Expect(gitDependencies).To(HaveLen(3))
				vcsGitDependencies := selectVcsGitDependencies(gitDependencies)
				Expect(vcsGitDependencies).To(HaveLen(2))
			})
		})

		Context("when I supply multiple artefacts files", func() {
			BeforeEach(func() {
				inputArtefactsPath1, err := filepath.Abs(filepath.Join("assets", "sources-file-multiple-archives.yml"))
				Expect(err).ToNot(HaveOccurred())
				inputArtefactsPath2, err := filepath.Abs(filepath.Join("assets", "sources-file-single-archive.yml"))
				Expect(err).ToNot(HaveOccurred())
				additionalArguments = []string{"--additional-sources-file", inputArtefactsPath1, "--additional-sources-file", inputArtefactsPath2}
			})

			It("adds multiple archiveDependency entries", func() {
				archiveDependencies := selectArchiveDependencies(metadataLabel.Dependencies)
				Expect(archiveDependencies).To(HaveLen(3))
			})
		})

		Context("when I supply erroneous paths as artefacts file", func(){
			It("exits with an error", func() {
				By("executing it")
				inputTarPath, err := filepath.Abs(filepath.Join("assets", "tiny.tgz"))
				Expect(err).ToNot(HaveOccurred())
				_, stdErr := runDepLab([]string{"--additional-sources-file", "erroneous_path.yml", "--image-tar", inputTarPath, "--git", pathToGitRepo}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("could not parse artefact file: erroneous_path.yml"))
			})
		})

		Context("when I supply empty file as artefacts file", func(){
			It("exits with an error", func() {
				By("executing it")
				inputTarPath, err := filepath.Abs(filepath.Join("assets", "tiny.tgz"))
				Expect(err).ToNot(HaveOccurred())
				inputArtefactsPath, err := filepath.Abs(filepath.Join("assets", "empty-file.yml"))
				Expect(err).ToNot(HaveOccurred())
				_, stdErr := runDepLab([]string{"--additional-sources-file", inputArtefactsPath, "--image-tar", inputTarPath, "--git", pathToGitRepo}, 1)
				errorOutput := strings.TrimSpace(string(getContentsOfReader(stdErr)))
				Expect(errorOutput).To(ContainSubstring("could not parse artefact file"))
			})
		})
	})


})

func selectVcsGitDependencies(dependencies []metadata.Dependency) []metadata.Dependency {
	var gitDependencies []metadata.Dependency
	for _, dependency := range dependencies {
		Expect(dependency.Source.Metadata).To(Not(BeNil()))
		gitSourceMetadata := dependency.Source.Metadata.(map[string]interface{})
		if dependency.Source.Type == "git" && gitSourceMetadata["url"].(string) != "https://example.com/example.git" {
			gitDependencies = append(gitDependencies, dependency)
		}
	}
	return gitDependencies
}