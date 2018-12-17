/*
Copyright 2018 Ridecell, Inc..

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package components_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	secrertsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	secretscomponents "github.com/Ridecell/ridecell-operator/pkg/controller/secrets/components"
)

var _ = Describe("PullSecret Defaults Component", func() {
	It("does nothing on a filled out object", func() {
		replicas := int32(2)
		instance := &secretsv1beta1.PullSecret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: secretsv1beta1.PullSecretSpec{
				PullSecret:            "foo-secret",
			},
		}
		ctx := &components.ComponentContext{Top: instance}

		comp := secretscomponents.NewDefaults()
		_, err := comp.Reconcile(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Spec.PullSecret).To(Equal("foo-secret"))
	})


	It("sets a default pull secret", func() {
		replicas := int32(2)
		instance := &secretsv1beta1.PullSecret{
			ObjectMeta: metav1.ObjectMeta{Name: "foo"},
			Spec: secretsv1beta1.PullSecretSpec{
			},
		}
		ctx := &components.ComponentContext{Top: instance}

		comp := secretscomponents.NewDefaults()
		_, err := comp.Reconcile(ctx)
		Expect(err).NotTo(HaveOccurred())
		Expect(instance.Spec.PullSecret).To(Equal("pull-secret"))
	})


})
