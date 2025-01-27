// Copyright (c) 2019-2020 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package integration_test

import (
	"github.com/vmware-tanzu/dependency-labeler/test/test_utils"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("deplab", func() {
	Context("when called with --metadata-file", func() {
		Describe("and metadata can be written", func() {
			It("succeeds", func() {
				metadataDestinationPath := test_utils.ExistingFileName()
				defer test_utils.CleanupFile(metadataDestinationPath)

				_, _ = runDepLab([]string{
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", metadataDestinationPath,
				}, 0)
			})
		})

		Describe("and metadata can't be written", func() {
			It("exits with 1 and throws an error about the file missing", func() {
				_, stdErr := runDepLab([]string{
					"--image-tar", getTestAssetPath("image-archives/tiny.tgz"),
					"--git", pathToGitRepo,
					"--metadata-file", "a-path-that-does-not-exist/foo.json",
				}, 1)

				Expect(string(getContentsOfReader(stdErr))).To(
					ContainSubstring("a-path-that-does-not-exist/foo.json"))
			})
		})
	})
})
