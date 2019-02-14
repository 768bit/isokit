// The Isomorphic Go Project
// Copyright (c) Wirecog, LLC. All rights reserved.
// Use of this source code is governed by a BSD-style
// license, which can be found in the LICENSE file.

package isokit

import (
	"bytes"
	"encoding/gob"
	"errors"
	"fmt"
	"github.com/768bit/packr"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var CogStaticAssetsSearchPaths []string

type TemplateSet struct {
	namespaces        map[string]*TemplateBundle
	members           map[string]*Template
	Funcs             template.FuncMap
	bundle            *TemplateBundle
	TemplateFilesPath string
	binaryBundle      []byte
}

func NewTemplateSet() *TemplateSet {
	return &TemplateSet{
		namespaces: map[string]*TemplateBundle{},
		members:    map[string]*Template{},
		Funcs:      template.FuncMap{},
	}
}

func (t *TemplateSet) Members() map[string]*Template {
	return t.members
}

func (t *TemplateSet) Bundle(namespace string) (*TemplateBundle, error) {
	if bundle, namespaceExists := t.namespaces[namespace]; !namespaceExists {
		return nil, errors.New(fmt.Sprintf("Cannot get bundle for namespace %s as it doesnt exist", namespace))
	} else {
		return bundle, nil
	}
}

func (t *TemplateSet) AddTemplateFile(namespace string, templateType TemplateType, templateName string, filename string) error {
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var tplstr = string(contents)
	tpl, err := template.New(templateName).Funcs(t.Funcs).Parse(tplstr)
	templateObj := Template{
		Template:     tpl,
		Namespace:    namespace,
		templateType: templateType,
	}

	if _, namespaceExists := t.namespaces[namespace]; !namespaceExists {
		t.namespaces[namespace] = NewTemplateBundle(namespace)
	}
	t.members[templateObj.NameWithNamespace()] = &templateObj
	t.namespaces[namespace].items[templateObj.NameWithPrefix()] = tplstr
	return nil

}

func (t *TemplateSet) MakeAllAssociations() error {

	for _, template := range t.members {

		for _, member := range t.members {

			if member.Lookup(template.NameWithNamespace()) == nil {

				if _, err := member.AddParseTree(template.NameWithNamespace(), template.Tree); err != nil {
					println(err)
					return err
				}
			}

		}

	}
	return nil
}

func (t *TemplateSet) ImportTemplatesFromMap(namespace string, bundle *TemplateBundle) error {

	for name, templateContents := range bundle.Items() {

		var templateType TemplateType
		if strings.HasPrefix(name, PrefixNamePartial) {
			templateType = TemplatePartial
		} else if strings.HasPrefix(name, PrefixNameView) {
			templateType = TemplateView
		} else if strings.HasPrefix(name, PrefixNameComponent) {
			templateType = TemplateComponent
		} else if strings.HasPrefix(name, PrefixNameDialog) {
			templateType = TemplateDialog
		} else if strings.HasPrefix(name, PrefixNameForm) {
			templateType = TemplateForm
		} else if strings.HasPrefix(name, PrefixNameLayout) {
			templateType = TemplateLayout
		} else {
			templateType = TemplateRegular
		}

		tpl, err := template.New(name).Funcs(t.Funcs).Parse(templateContents)

		if err != nil {
			log.Println(templateContents)
			log.Println("Encountered error when attempting to parse template: ", err)

			return err
		}

		tmpl := Template{
			Namespace:    namespace,
			Template:     tpl,
			templateType: templateType,
		}
		t.members[namespace+"/"+name] = &tmpl

		if _, namespaceExists := t.namespaces[namespace]; !namespaceExists {
			t.namespaces[namespace] = bundle
		} else {
			t.namespaces[namespace].addItems(bundle.Items())
		}

	}
	t.MakeAllAssociations()
	return nil
}

func (t *TemplateSet) Render(templateURI string, params *RenderParams) error {

	return t.Members()[templateURI].Render(params)

}

func (t *TemplateSet) RenderSimple(templateURI string, params interface{}) error {

	return t.Members()[templateURI].RenderSimple(params)

}

func (t *TemplateSet) PersistTemplateBundleToDisk(bundlePath string) error {

	dirPath := filepath.Dir(bundlePath)
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {

		return errors.New("The specified directory for the template bundle, " + dirPath + ", does not exist!")

	} else {

		var templateContentItemsBuffer bytes.Buffer
		enc := gob.NewEncoder(&templateContentItemsBuffer)
		nmap := map[string]map[string]string{}
		for namespace, bundle := range t.namespaces {
			nmap[namespace] = bundle.Items()
		}
		err := enc.Encode(&nmap)
		if err != nil {
			return err
		}

		t.binaryBundle = templateContentItemsBuffer.Bytes()

		err = ioutil.WriteFile(bundlePath, t.binaryBundle, 0644)
		if err != nil {
			return err
		} else {
			return nil
		}

	}

}

func (t *TemplateSet) RestoreTemplateBundleFromDisk(bundlePath string) error {

	if _, err := os.Stat(bundlePath); os.IsNotExist(err) {
		return errors.New("The bundle path, " + bundlePath + ", does not exist")
	} else {

		data, err := ioutil.ReadFile(bundlePath)
		if err != nil {
			return err
		}

		return t.RestoreTemplateBundleFromBinary(data)
	}
}

func (t *TemplateSet) RestoreTemplateBundleFromBinary(bundle []byte) error {

	var templateBundleMap map[string]map[string]string
	var templateBundleMapBuffer bytes.Buffer
	dec := gob.NewDecoder(&templateBundleMapBuffer)
	templateBundleMapBuffer = *bytes.NewBuffer(bundle)
	err := dec.Decode(&templateBundleMap)

	if err != nil {
		return err
	}

	t.binaryBundle = templateBundleMapBuffer.Bytes()

	for namespace, templateMap := range templateBundleMap {
		bnd := NewTemplateBundle(namespace)
		bnd.addItems(templateMap)
		err = t.ImportTemplatesFromMap(namespace, bnd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (t *TemplateSet) GetTemplateBundleBinary() []byte {

	return t.binaryBundle

}

func (t *TemplateSet) GatherTemplatesFromPath(namespace string, templatesPath string) error {

	bundle := NewTemplateBundle(namespace)

	bundle.importTemplateFileContents(templatesPath)
	return t.ImportTemplatesFromMap(namespace, bundle)

}

func (t *TemplateSet) GatherTemplatesFromPackrBox(namespace string, box *packr.Box, path string) error {

	bundle := NewTemplateBundle(namespace)

	bundle.importTemplateFileContentsFromBox(box, path)
	return t.ImportTemplatesFromMap(namespace, bundle)

}
