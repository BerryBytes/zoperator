/*
Copyright 2024.

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

package controller

import (
	"context"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"

	sealedsecretsv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"
)

var _ = Describe("UserConfig Controller", func() {
	const timeout = time.Second * 30
	const interval = time.Second * 1

	var resourceName, namespace string
	var typeNamespacedName types.NamespacedName
	var userconfig *myoperatorv1alpha1.UserConfig

	BeforeEach(func() {
		// failed test runs that don't clean up leave resources behind.
		ctx = context.Background()
		if resourceName != "" {
			By("cleaning up any existing resources")
			existingConfig := &myoperatorv1alpha1.UserConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
			}
			err := k8sClient.Delete(ctx, existingConfig)
			if err != nil {
				if !errors.IsNotFound(err) {
					Expect(err).NotTo(HaveOccurred())
				}
			}
			time.Sleep(time.Second * 5)
		}

		randomVal := fmt.Sprintf("%d", GinkgoRandomSeed())
		resourceName = "test-user-" + randomVal
		namespace = "test-user-" + randomVal + "-namespace"
		// Create new resources
		userconfig = &myoperatorv1alpha1.UserConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      resourceName,
				Namespace: namespace,
			},
			Spec: myoperatorv1alpha1.UserConfigSpec{
				Identity: myoperatorv1alpha1.Identity{
					Username: "test-user",
					Contact:  "test@example.com",
					Groups:   []string{"viewer"},
				},
				Permissions: myoperatorv1alpha1.Permissions{
					Resources: []myoperatorv1alpha1.ResourcePermission{
						{
							Resource:  "pods",
							Operation: "viewer",
						},
					},
				},
				Secrets: []myoperatorv1alpha1.Secret{
					{
						Name: "test-sealed-secret",
						Type: "sealed",
						SealedSecret: &myoperatorv1alpha1.SealedSecret{
							EncryptedData: map[string]string{
								"password": "AgCCCQB6Isjzt/mdp3+n7C57RG0aBLwU/0b8DSYCIPJgPKn/il9C7ZPVD6r2GiD1u2GHYhva01wLNUnU+AnB6ZGL5oD1kiJM+FtGOMywsEySC/JTveOvDOPavwx5JGpd1vtTjPBnf2w7bSCwXyO0bGKLQOWjJsKWK/UQ9zeZ+TAa5TYWRkj9Je2sl2jA0uCt2vp1srY9AjvsQoTYYT6SZ27tLyzkCpZkwLCqOTtgq+kvb+jltmltnflx/7RNBX5+5a9JfrbT2tn24CASpANnqs9lxkrjcfhhfIAy9wM/ffmdaKc5PQjTRowcWu//nkRL2iTTJSd8rB8EssLcpxngywmWaFBJ9vnbMiQQ0F7seeSBTI+pGk4HhmTvRV2I8IYFXfZQWMr2bw9DXBAUoyaVHPM4EWfme90+iDmrkSYS6E9++sHTHxqIswoi04ngS0VYDiJ8YWSR9ulTPJ6A0jOJYsUI6sVRCwMDwa9k1/E3nk3SnIlzgXXwrP+kvAqBfMjvlFwdygFxlIz2NEyajqVLezsoyCDMxHNnYt2dIR1Ji1URznbYf2jvLDjhIt0WXJ6t7XJqUrJmL2wGm7gznPSlkp/iSTHFoFge4B5T4iZ+6uMUgjJYorCJpHmheBSKg3/i/ILm2EPv6VnQGgk434QwbbeGh+zh1LBN3XEKXq0TAjOmZU9rBFdGG6M1IIJzNEL4x472z61Gh6ut+xRC",
							},
						},
					},
				},
			},
		}
		Expect(k8sClient.Create(ctx, userconfig)).To(Succeed())

		typeNamespacedName = types.NamespacedName{
			Name:      resourceName,
			Namespace: namespace,
		}
		// Wait for creation to complete
		Eventually(func() error {
			return k8sClient.Get(ctx, typeNamespacedName, userconfig)
		}, "10s", "1s").Should(Succeed())
	})

	Context("Creating a basic UserConfig", func() {
		It("should successfully create the resource", func() {
			resource := &myoperatorv1alpha1.UserConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Spec).To(Equal(userconfig.Spec))
			GinkgoWriter.Printf("Resource Created =====: %v\n", resource)
		})

		It("should create a namespace when UserConfig is created", func() {
			// First reconcile
			controllerReconciler := &UserConfigReconciler{
				Client: k8sManager.GetClient(),
				Scheme: k8sManager.GetScheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())
			namespace := &corev1.Namespace{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: typeNamespacedName.Namespace}, namespace)
			Expect(err).NotTo(HaveOccurred())
		})
		It("should update the resource when UserConfig is updated", func() {
			// get the config and update it
			var userconfig myoperatorv1alpha1.UserConfig
			Expect(k8sClient.Get(ctx, typeNamespacedName, &userconfig)).To(Succeed())
			userconfig.Spec.Identity.Username = "updated-user"
			Expect(k8sClient.Update(ctx, &userconfig)).To(Succeed())

			resource := &myoperatorv1alpha1.UserConfig{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())
			Expect(resource.Spec).To(Equal(userconfig.Spec))
		})
		It("should delete the namespace when UserConfig is deleted", func() {
			userconfig := &myoperatorv1alpha1.UserConfig{
				ObjectMeta: metav1.ObjectMeta{
					Name:      resourceName,
					Namespace: namespace,
				},
			}
			Expect(k8sClient.Delete(ctx, userconfig)).To(Succeed())
		})
	})
	Context("Creating a UserConfig with a sealed secret", func() {
		It("should create a sealed-secret when UserConfig is created", func() {
			// First reconcile
			controllerReconciler := &UserConfigReconciler{
				Client: k8sManager.GetClient(),
				Scheme: k8sManager.GetScheme(),
			}
			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).To(HaveOccurred())

			// Then check sealed secret
			secret := &sealedsecretsv1alpha1.SealedSecret{}
			err = k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-sealed-secret",
				Namespace: namespace,
			}, secret)
			GinkgoWriter.Printf("Sealed Secret Created, %v\n", secret)
			Expect(err).NotTo(HaveOccurred())
		})

	})
})
