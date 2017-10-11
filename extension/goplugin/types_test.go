// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package goplugin_test

import (
	"encoding/json"

	"github.com/cloudwan/gohan/extension/goext"
	"github.com/cloudwan/gohan/extension/goplugin"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Types tests", func() {
	var (
		env *goplugin.Environment
	)

	BeforeEach(func() {
		env = goplugin.NewEnvironment("test", nil, nil)
	})

	AfterEach(func() {
		env.Stop()
	})

	Describe("JSON Marshalling", func() {
		Context("String", func() {
			type TestResource struct {
				Value goext.MaybeString `json:"value"`
			}
			It("value defined", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeMaybeString("hello")})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":\"hello\"}"))
			})

			It("value undefined", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullString()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})
		})

		Context("Float", func() {
			type TestResource struct {
				Value goext.MaybeFloat `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeMaybeFloat(1.23)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":1.23}"))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullFloat()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})
		})

		Context("Bool", func() {
			type TestResource struct {
				Value goext.MaybeBool `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeMaybeBool(true)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":true}"))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullBool()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})
		})

		Context("Int", func() {
			type TestResource struct {
				Value goext.MaybeInt `json:"value"`
			}
			It("defined value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeMaybeInt(123)})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":123}"))
			})

			It("undefined value", func() {
				buf, err := json.Marshal(TestResource{})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})

			It("null value", func() {
				buf, err := json.Marshal(TestResource{Value: goext.MakeNullInt()})
				Expect(err).To(BeNil())
				Expect(string(buf)).To(Equal("{\"value\":null}"))
			})
		})

	})
})
