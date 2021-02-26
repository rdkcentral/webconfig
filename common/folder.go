/**
* Copyright 2021 Comcast Cable Communications Management, LLC
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
* http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
* SPDX-License-Identifier: Apache-2.0
*/
package common

type Folder struct {
	docmap map[string]Document
}

func NewFolder() *Folder {
	docmap := map[string]Document{}
	return &Folder{
		docmap: docmap,
	}
}

func (f *Folder) SetDocument(groupId string, d *Document) {
	f.docmap[groupId] = *d
}

func (f *Folder) Document(groupId string) *Document {
	d, ok := f.docmap[groupId]
	if !ok {
		return nil
	}
	return &d
}

func (f *Folder) DeleteDocument(groupId string) {
	delete(f.docmap, groupId)
}

func (f *Folder) VersionMap() map[string]string {
	versionMap := map[string]string{}
	for k, doc := range f.docmap {
		if doc.Version() != nil {
			versionMap[k] = *doc.Version()
		}
	}
	return versionMap
}

func (f *Folder) Length() int {
	return len(f.docmap)
}

func (f *Folder) Items() map[string]Document {
	return f.docmap
}
