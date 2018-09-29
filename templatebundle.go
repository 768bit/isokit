// The Isomorphic Go Project
// Copyright (c) Wirecog, LLC. All rights reserved.
// Use of this source code is governed by a BSD-style
// license, which can be found in the LICENSE file.

package isokit

import (
	"fmt"
	"github.com/gobuffalo/packr"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type TemplateBundle struct {
	namespace string
	items     map[string]string
}

func NewTemplateBundle(namespace string) *TemplateBundle {

	return &TemplateBundle{
		namespace: namespace,
		items:     map[string]string{},
	}

}

func (t *TemplateBundle) Items() map[string]string {
	return t.items
}

func normalizeTemplateNameForWindows(path, templateDirectory, TemplateFileExtension string) string {

	result := strings.Replace(path, templateDirectory, "", -1)
	result = strings.Replace(result, string(os.PathSeparator), "/", -1)
	result = strings.Replace(result, TemplateFileExtension, "", -1)
	result = strings.TrimPrefix(result, `/`)
	return result
}

func normalizeCogTemplateNameForWindows(path, templateDirectory, TemplateFileExtension string) string {

	result := strings.Replace(path, templateDirectory, "", -1)
	result = strings.Replace(result, string(os.PathSeparator), "/", -1)
	result = strings.Replace(result, TemplateFileExtension, "", -1)
	result = strings.TrimPrefix(result, `/`)
	result = result + "/" + result
	return result
}

func (t *TemplateBundle) addItems(items map[string]string) {

	for key, item := range items {
		t.items[key] = item
	}

}

func (t *TemplateBundle) importTemplateFileContents(templatesPath string) error {

	templateDirectory := filepath.Clean(templatesPath)

	fmt.Printf("Importing Templates from: %s\n", templatesPath)

	if err := filepath.Walk(templateDirectory, func(path string, info os.FileInfo, err error) error {
		fmt.Printf("Checking Template Path: %s\n", path)
		if strings.HasSuffix(path, TemplateFileExtension) {
			name := strings.TrimSuffix(strings.TrimPrefix(path, templateDirectory+"/"), TemplateFileExtension)

			if runtime.GOOS == "windows" {
				name = normalizeTemplateNameForWindows(path, templateDirectory, TemplateFileExtension)
			}

			contents, err := ioutil.ReadFile(path)
			t.items[name] = string(contents)

			if err != nil {
				log.Println("error encountered while walking directory: ", err)
				return err
			}

		}
		return nil
	}); err != nil {
		return err
	}

	return nil

}

func (t *TemplateBundle) importTemplateFileContentsFromBox(box *packr.Box, templatesPath string) error {

	prefixPath := templatesPath

	if prefixPath != "" {
		prefixPath += "/"
	}

	if err := box.WalkPrefix(templatesPath, func(path string, file packr.File) error {

		if strings.HasSuffix(path, TemplateFileExtension) {
			name := strings.TrimSuffix(strings.TrimPrefix(path, prefixPath), TemplateFileExtension)

			contents, err := ioutil.ReadAll(file)
			if err != nil {
				log.Println("error encountered while walking directory: ", err)
				return err
			}

			t.items[name] = string(contents)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil

}
