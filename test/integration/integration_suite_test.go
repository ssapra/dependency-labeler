// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vmware-tanzu/dependency-labeler/pkg/metadata"

	"github.com/onsi/gomega/ghttp"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing/object"

	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var (
	pathToBin                 string
	commitHash, pathToGitRepo string
)

func TestDeplab(t *testing.T) {
	RegisterFailHandler(Fail)

	BeforeSuite(func() {
		var (
			err error
		)

		commitHash, pathToGitRepo = makeFakeGitRepo()

		pathToBin, err = gexec.Build("github.com/vmware-tanzu/dependency-labeler/cmd/deplab")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterSuite(func() {
		os.RemoveAll(pathToGitRepo)
		gexec.Kill()
		gexec.CleanupBuildArtifacts()
	})

	RunSpecs(t, "deplab Suite")
}

func runDepLab(args []string, expErrCode int) (stdOut *bytes.Reader, stdErr *bytes.Reader) {
	stdOutBuffer := bytes.Buffer{}
	stdErrBuffer := bytes.Buffer{}

	cmd := exec.Command(pathToBin, args...)

	session, err := gexec.Start(cmd, &stdOutBuffer, &stdErrBuffer)
	Expect(err).ToNot(HaveOccurred())
	<-session.Exited

	stdOut = bytes.NewReader(stdOutBuffer.Bytes())
	stdErr = bytes.NewReader(stdErrBuffer.Bytes())

	if os.Getenv("DEBUG") != "" {
		io.Copy(os.Stdout, stdOut)
		io.Copy(os.Stdout, stdErr)
		stdOut.Seek(0, 0)
		stdErr.Seek(0, 0)
	}

	Eventually(session, time.Minute).Should(gexec.Exit(expErrCode))

	return stdOut, stdErr
}

func runDeplabAgainstImage(inputImage string, extraArgs ...string) (metadataLabel metadata.Metadata) {
	f, err := ioutil.TempFile("", "")
	Expect(err).ToNot(HaveOccurred())

	defer func() {
		Expect(os.Remove(f.Name())).ToNot(HaveOccurred())
	}()

	By("executing it")
	args := []string{"--image", inputImage, "--git", pathToGitRepo, "--metadata-file", f.Name()}
	args = append(args, extraArgs...)
	_, _ = runDepLab(args, 0)

	metadataLabel = metadata.Metadata{}
	err = json.NewDecoder(f).Decode(&metadataLabel)
	Expect(err).ToNot(HaveOccurred())

	return metadataLabel
}

func runDeplabAgainstTar(inputTarPath string, extraArgs ...string) (metadataLabel metadata.Metadata) {
	metadataLabel, _ = runDeplabAgainstTarReportErrorMessages(inputTarPath, extraArgs...)

	return metadataLabel
}

func runDeplabAgainstTarReportErrorMessages(inputTarPath string, extraArgs ...string) (metadataLabel metadata.Metadata, errorOutput string) {
	f, err := ioutil.TempFile("", "")
	Expect(err).ToNot(HaveOccurred())
	defer func() {
		Expect(os.Remove(f.Name())).ToNot(HaveOccurred())
	}()

	By("executing it")
	args := []string{"--image-tar", inputTarPath, "--git", pathToGitRepo, "--metadata-file", f.Name()}
	args = append(args, extraArgs...)
	_, stdErr := runDepLab(args, 0)

	decoder := json.NewDecoder(f)
	metadataLabel = metadata.Metadata{}
	err = decoder.Decode(&metadataLabel)
	Expect(err).ToNot(HaveOccurred())

	errorOutput = strings.TrimSpace(string(getContentsOfReader(stdErr)))

	return metadataLabel, errorOutput
}

func makeFakeGitRepo() (string, string) {
	path, err := ioutil.TempDir("", "deplab-integration")
	Expect(err).ToNot(HaveOccurred())

	repo, err := git.PlainInit(path, false)
	Expect(err).ToNot(HaveOccurred())

	_, err = repo.CreateRemote(&config.RemoteConfig{
		Name: "origin",
		URLs: []string{"https://example.com/example.git"},
	})
	Expect(err).ToNot(HaveOccurred())

	testFilePath := filepath.Join(path, "test")
	data := []byte("TestFile\n")
	err = ioutil.WriteFile(testFilePath, data, 0644)
	Expect(err).ToNot(HaveOccurred())

	w, err := repo.Worktree()
	Expect(err).ToNot(HaveOccurred())

	err = w.AddGlob("*")
	Expect(err).ToNot(HaveOccurred())

	ch, err := w.Commit("Test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Pivotal Example",
			Email: "example@pivotal.io",
			When:  time.Now(),
		},
	})
	Expect(err).ToNot(HaveOccurred())

	repo.CreateTag("foo", ch, nil)

	ch, err = w.Commit("Second test commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Pivotal Example",
			Email: "example@pivotal.io",
			When:  time.Now(),
		},
	})
	Expect(err).ToNot(HaveOccurred())

	repo.CreateTag("bar", ch, nil)

	return ch.String(), path
}

func getContentsOfReader(r io.Reader) []byte {
	contents, err := ioutil.ReadAll(r)
	Expect(err).NotTo(HaveOccurred())

	return contents
}
func startServer(handlers ...http.HandlerFunc) (server *ghttp.Server) {
	server = ghttp.NewServer()
	server.AppendHandlers(
		handlers...,
	)
	return server
}

func getTestAssetPath(path string) string {
	inputTarPath := filepath.Join("assets", path)
	inputTarPath, err := filepath.Abs(inputTarPath)
	if err != nil {
		log.Fatalf("Could not find test asset %s: %s", path, err)
	}
	return inputTarPath
}
