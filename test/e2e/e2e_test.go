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

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/onsi/gomega/types"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
	"01cloud/zoperator/test/utils"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// namespace where the project is deployed in
const namespace = "lab-system"

// serviceAccountName created for the project
const serviceAccountName = "lab-controller-manager"

// metricsServiceName is the name of the metrics service of the project
const metricsServiceName = "lab-controller-manager-metrics-service"

// metricsRoleBindingName is the name of the RBAC that will be created to allow get the metrics data
const metricsRoleBindingName = "lab-metrics-binding"

var (
	controllerPodName string
	k8sClient         client.Client
)

var _ = Describe("Manager", Ordered, func() {
	// Before running the tests, set up the environment by creating the namespace,
	// installing CRDs, deploying the controller, and initializing the Kubernetes client.
	BeforeAll(func() {
		By("creating manager namespace")
		cmd := exec.Command("kubectl", "create", "ns", namespace)
		_, err := utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create namespace")

		By("making CRDs")
		cmd = exec.Command("make", "manifests")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to make manifests")

		By("installing Sealed secrets CRDs")
		cmd = exec.Command("kubectl", "apply", "-f", filepath.Join("config", "crd", "sealed-secrets"))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install sealedsecrets CRDs")

		By("installing CRDs")
		cmd = exec.Command("make", "install")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to install CRDs")

		By("verifying UserConfig CRD is available")
		verifyCRDReady := func(g Gomega) {
			cmd := exec.Command("kubectl", "get", "crd", "userconfigs.myoperator.01cloud.io")
			output, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to get UserConfig CRD")
			g.Expect(output).To(ContainSubstring("userconfigs.myoperator.01cloud.io"))

			cmd = exec.Command("kubectl", "get", "crd", "userconfigs.myoperator.01cloud.io", "-o", "jsonpath={.status.conditions[?(@.type=='Established')].status}")
			status, err := utils.Run(cmd)
			g.Expect(err).NotTo(HaveOccurred(), "Failed to check CRD Established condition")
			g.Expect(status).To(Equal("True"), "UserConfig CRD is not established")
		}
		Eventually(verifyCRDReady, 30*time.Second, time.Second).Should(Succeed())

		By("creating secret")
		cmd = exec.Command("kubectl", "create", "secret", "generic", "kubeconfig-secret", "--from-file=config=/tmp/zoperator-test-cluster", "-n", "lab-system")
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to create secret")

		By("deploying the controller-manager")
		cmd = exec.Command("make", "deploy", fmt.Sprintf("IMG=%s", projectImage))
		_, err = utils.Run(cmd)
		Expect(err).NotTo(HaveOccurred(), "Failed to deploy the controller-manager")

		// Initialize the Kubernetes client
		By("initializing Kubernetes client")
		var config *rest.Config
		kubeconfigPath := os.Getenv("KUBECONFIG")
		if kubeconfigPath == "" {
			kubeconfigPath = "/tmp/zoperator-test-cluster"
			By("KUBECONFIG not set, defaulting to " + kubeconfigPath)
		}

		// Check if the kubeconfig file exists
		if _, err := os.Stat(kubeconfigPath); os.IsNotExist(err) {
			Fail(fmt.Sprintf("Kubeconfig file not found at %s. Please set the KUBECONFIG environment variable or ensure the file exists.", kubeconfigPath))
		}

		config, err = clientcmd.BuildConfigFromFlags("", kubeconfigPath)
		if err != nil {
			Fail(fmt.Sprintf("Failed to build config from kubeconfig at %s: %v", kubeconfigPath, err))
		}

		// Create a new scheme and register the UserConfig type
		s := runtime.NewScheme()
		err = myoperatorv1alpha1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred(), "Failed to register UserConfig scheme")

		// Create the client with the registered scheme
		k8sClient, err = client.New(config, client.Options{Scheme: s})
		Expect(err).NotTo(HaveOccurred(), "Failed to create Kubernetes client")

		// Register corev1 (for Namespace, ResourceQuota, LimitRange, ServiceAccount, Secret)
		err = corev1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred(), "Failed to register corev1 scheme")

		// Register rbacv1 (for Role, RoleBinding)
		err = rbacv1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred(), "Failed to register rbacv1 scheme")

		// Register networkingv1 (for NetworkPolicy)
		err = networkingv1.AddToScheme(s)
		Expect(err).NotTo(HaveOccurred(), "Failed to register networkingv1 scheme")

	})

	// After all tests have been executed, clean up by undeploying the controller, uninstalling CRDs,
	// and deleting the namespace.
	AfterAll(func() {
		By("cleaning up the UserConfig resource")
		cmd := exec.Command("kubectl", "delete", "userconfig", "test-user", "--ignore-not-found")
		_, _ = utils.Run(cmd)

		By("cleaning up the curl pod for metrics")
		cmd = exec.Command("kubectl", "delete", "pod", "curl-metrics", "-n", namespace)
		_, _ = utils.Run(cmd)

		By("undeploying the controller-manager")
		cmd = exec.Command("make", "undeploy")
		_, _ = utils.Run(cmd)

		By("uninstalling CRDs")
		cmd = exec.Command("make", "uninstall")
		_, _ = utils.Run(cmd)

		By("removing manager namespace")
		cmd = exec.Command("kubectl", "delete", "ns", namespace)
		_, _ = utils.Run(cmd)
	})

	// After each test, check for failures and collect logs, events,
	// and pod descriptions for debugging.
	AfterEach(func() {
		specReport := CurrentSpecReport()
		if specReport.Failed() {
			By("Fetching controller manager pod logs")
			cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
			controllerLogs, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Controller logs:\n %s", controllerLogs)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Controller logs: %s", err)
			}

			By("Fetching Kubernetes events")
			cmd = exec.Command("kubectl", "get", "events", "-n", namespace, "--sort-by=.lastTimestamp")
			eventsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Kubernetes events:\n%s", eventsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get Kubernetes events: %s", err)
			}

			By("Fetching curl-metrics logs")
			cmd = exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
			metricsOutput, err := utils.Run(cmd)
			if err == nil {
				_, _ = fmt.Fprintf(GinkgoWriter, "Metrics logs:\n %s", metricsOutput)
			} else {
				_, _ = fmt.Fprintf(GinkgoWriter, "Failed to get curl-metrics logs: %s", err)
			}

			By("Fetching controller manager pod description")
			cmd = exec.Command("kubectl", "describe", "pod", controllerPodName, "-n", namespace)
			podDescription, err := utils.Run(cmd)
			if err == nil {
				fmt.Println("Pod description:\n", podDescription)
			} else {
				fmt.Println("Failed to describe controller pod")
			}
		}
	})

	SetDefaultEventuallyTimeout(2 * time.Minute)
	SetDefaultEventuallyPollingInterval(time.Second)

	Context("Manager", func() {
		It("should run successfully", func() {
			By("validating that the controller-manager pod is running as expected")
			verifyControllerUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get",
					"pods", "-l", "control-plane=controller-manager",
					"-o", "go-template={{ range .items }}"+
						"{{ if not .metadata.deletionTimestamp }}"+
						"{{ .metadata.name }}"+
						"{{ \"\\n\" }}{{ end }}{{ end }}",
					"-n", namespace,
				)

				podOutput, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to retrieve controller-manager pod information")
				podNames := utils.GetNonEmptyLines(podOutput)
				g.Expect(podNames).To(HaveLen(1), "expected 1 controller pod running")
				controllerPodName = podNames[0]
				g.Expect(controllerPodName).To(ContainSubstring("controller-manager"))

				cmd = exec.Command("kubectl", "get",
					"pods", controllerPodName, "-o", "jsonpath={.status.phase}",
					"-n", namespace,
				)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Running"), "Incorrect controller-manager pod status")
			}
			Eventually(verifyControllerUp).Should(Succeed())
		})

		It("should ensure the metrics endpoint is serving metrics", func() {
			By("creating a ClusterRoleBinding for the service account to allow access to metrics")
			cmd := exec.Command("kubectl", "get", "clusterrolebinding", metricsRoleBindingName)
			_, err := utils.Run(cmd)
			if err == nil {
				By("deleting existing ClusterRoleBinding")
				cmd = exec.Command("kubectl", "delete", "clusterrolebinding", metricsRoleBindingName)
				_, err = utils.Run(cmd)
				Expect(err).NotTo(HaveOccurred(), "Failed to delete existing ClusterRoleBinding")
			}

			cmd = exec.Command("kubectl", "create", "clusterrolebinding", metricsRoleBindingName,
				"--clusterrole=lab-metrics-reader",
				fmt.Sprintf("--serviceaccount=%s:%s", namespace, serviceAccountName),
			)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create ClusterRoleBinding")

			By("validating that the metrics service is available")
			cmd = exec.Command("kubectl", "get", "service", metricsServiceName, "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Metrics service should exist")

			By("validating that the ServiceMonitor for Prometheus is applied in the namespace")
			cmd = exec.Command("kubectl", "get", "ServiceMonitor", "-n", namespace)
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "ServiceMonitor should exist")

			By("getting the service account token")
			token, err := serviceAccountToken()
			Expect(err).NotTo(HaveOccurred())
			Expect(token).NotTo(BeEmpty())

			By("waiting for the metrics endpoint to be ready")
			verifyMetricsEndpointReady := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "endpoints", metricsServiceName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("8443"), "Metrics endpoint is not ready")
			}
			Eventually(verifyMetricsEndpointReady).Should(Succeed())

			By("verifying that the controller manager is serving the metrics server")
			verifyMetricsServerStarted := func(g Gomega) {
				cmd := exec.Command("kubectl", "logs", controllerPodName, "-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(ContainSubstring("controller-runtime.metrics\tServing metrics server"),
					"Metrics server not yet started")
			}
			Eventually(verifyMetricsServerStarted).Should(Succeed())

			By("creating the curl-metrics pod to access the metrics endpoint")
			cmd = exec.Command("kubectl", "run", "curl-metrics", "--restart=Never",
				"--namespace", namespace,
				"--image=curlimages/curl:7.78.0",
				"--", "/bin/sh", "-c", fmt.Sprintf(
					"curl -v -k -H 'Authorization: Bearer %s' https://%s.%s.svc.cluster.local:8443/metrics",
					token, metricsServiceName, namespace))
			_, err = utils.Run(cmd)
			Expect(err).NotTo(HaveOccurred(), "Failed to create curl-metrics pod")

			By("waiting for the curl-metrics pod to complete")
			verifyCurlUp := func(g Gomega) {
				cmd := exec.Command("kubectl", "get", "pods", "curl-metrics",
					"-o", "jsonpath={.status.phase}",
					"-n", namespace)
				output, err := utils.Run(cmd)
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("Succeeded"), "curl pod in wrong status")
			}
			Eventually(verifyCurlUp, 5*time.Minute).Should(Succeed())

			By("getting the metrics by checking curl-metrics logs")
			metricsOutput := getMetricsOutput()
			Expect(metricsOutput).To(ContainSubstring("controller_runtime_reconcile_total"))
		})
	})

	Context("When creating a UserConfig resource", func() {
		It("should create the expected resources", func() {
			By("Creating a UserConfig resource via API")
			testUserConfig := &myoperatorv1alpha1.UserConfig{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "myoperator.01cloud.io/v1alpha1",
					Kind:       "UserConfig",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-user",
				},
				Spec: myoperatorv1alpha1.UserConfigSpec{
					Identity: myoperatorv1alpha1.Identity{
						Username: "testuser",
						Contact:  "test@example.com",
						Groups:   []string{"viewer"},
					},
					Permissions: myoperatorv1alpha1.Permissions{
						Resources: []myoperatorv1alpha1.ResourcePermission{
							{
								Resource:  "pods",
								Operation: "CRUD",
							},
						},
					},
					ResourceQuotas: &myoperatorv1alpha1.ResourceQuota{
						Pods: "5",
						CPU:  "1",
					},
					LimitRange: &myoperatorv1alpha1.LimitRange{
						Limits: []myoperatorv1alpha1.LimitRangeLimit{
							{
								Type: "Container",
								Max: &myoperatorv1alpha1.Resources{
									CPU:    "1",
									Memory: "1Gi",
								},
								Min: &myoperatorv1alpha1.Resources{
									CPU:    "100m",
									Memory: "128Mi",
								},
							},
						},
					},
				},
			}

			err := k8sClient.Create(context.Background(), testUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to create UserConfig resource via API")

			userConfigNamespace := fmt.Sprintf("%s-namespace", testUserConfig.Name)

			By("Verifying the UserConfig resource is created")
			Eventually(func(g Gomega) {
				createdUserConfig := &myoperatorv1alpha1.UserConfig{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, createdUserConfig)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get UserConfig resource")
				g.Expect(createdUserConfig.Spec.Identity.Username).To(Equal("testuser"))
			}, 60*time.Second, time.Second).Should(Succeed())

			By("Verifying the UserConfig status is updated")
			Eventually(func(g Gomega) {
				updatedUserConfig := &myoperatorv1alpha1.UserConfig{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get UserConfig status")
				g.Expect(updatedUserConfig.Status.Conditions).To(HaveLen(2), "Status conditions should be present")
				g.Expect(updatedUserConfig.Status.Conditions[0].Status).To(Equal(metav1.ConditionTrue), "UserConfig status should be True")
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the UserConfig resource is reconciled")
			Eventually(func(g Gomega) {
				updatedUserConfig := &myoperatorv1alpha1.UserConfig{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get UserConfig condition")
				g.Expect(updatedUserConfig.Status.Conditions).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Type":   Equal("Ready"),
					"Reason": Equal("Reconciled"),
				})))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the Namespace is created")
			Eventually(func(g Gomega) {
				ns := &corev1.Namespace{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: userConfigNamespace}, ns)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get Namespace")
				g.Expect(ns.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "userconfig-operator"))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the ResourceQuota is created")
			Eventually(func(g Gomega) {
				resourceQuota := &corev1.ResourceQuota{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "default-resource-quota"}, resourceQuota)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get ResourceQuota")
				g.Expect(resourceQuota.Spec.Hard).To(HaveKeyWithValue(corev1.ResourceName("pods"), EqualQuantity("5")))
				g.Expect(resourceQuota.Spec.Hard).To(HaveKeyWithValue(corev1.ResourceName("cpu"), EqualQuantity("1")))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the LimitRange is created")
			Eventually(func(g Gomega) {
				limitRange := &corev1.LimitRange{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-limit-range"}, limitRange)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get LimitRange")
				g.Expect(limitRange.Spec.Limits).To(HaveLen(1))
				limit := limitRange.Spec.Limits[0]
				g.Expect(limit.Type).To(Equal(corev1.LimitTypeContainer))
				g.Expect(limit.Max.Cpu().String()).To(Equal("1"))
				g.Expect(limit.Max.Memory().String()).To(Equal("1Gi"))
				g.Expect(limit.Min.Cpu().String()).To(Equal("100m"))
				g.Expect(limit.Min.Memory().String()).To(Equal("128Mi"))
				g.Expect(limit.Default.Cpu().String()).To(Equal("500m"))
				g.Expect(limit.Default.Memory().String()).To(Equal("1Gi"))
				g.Expect(limit.DefaultRequest.Cpu().String()).To(Equal("250m"))
				g.Expect(limit.DefaultRequest.Memory().String()).To(Equal("512Mi"))
			}, 120*time.Second, time.Second).Should(Succeed())

			By("Verifying the Role is created")
			Eventually(func(g Gomega) {
				role := &rbacv1.Role{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-role"}, role)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get Role")
				g.Expect(role.Rules).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Resources": ContainElement("pods"),
					"Verbs":     ContainElements("create", "get", "list", "watch", "update", "patch", "delete"),
				})))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the ServiceAccount is created")
			Eventually(func(g Gomega) {
				sa := &corev1.ServiceAccount{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-serviceaccount"}, sa)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get ServiceAccount")
				g.Expect(sa.Labels).To(HaveKeyWithValue("app.kubernetes.io/managed-by", "userconfig-operator"))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the RoleBinding is created")
			Eventually(func(g Gomega) {
				roleBinding := &rbacv1.RoleBinding{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-rolebinding"}, roleBinding)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get RoleBinding")
				g.Expect(roleBinding.Subjects).To(ContainElements(
					MatchFields(IgnoreExtras, Fields{
						"Kind": Equal("User"),
						"Name": Equal("test-user"),
					}),
					MatchFields(IgnoreExtras, Fields{
						"Kind":      Equal("ServiceAccount"),
						"Name":      Equal("test-user-serviceaccount"),
						"Namespace": Equal(userConfigNamespace),
					}),
				))
				g.Expect(roleBinding.RoleRef).To(MatchFields(IgnoreExtras, Fields{
					"Kind":     Equal("Role"),
					"Name":     Equal("test-user-role"),
					"APIGroup": Equal("rbac.authorization.k8s.io"),
				}))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the NetworkPolicy is created")
			Eventually(func(g Gomega) {
				netpol := &networkingv1.NetworkPolicy{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-network-policy"}, netpol)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get NetworkPolicy")
				g.Expect(netpol.Spec.PolicyTypes).To(ContainElements(networkingv1.PolicyTypeIngress, networkingv1.PolicyTypeEgress))
				g.Expect(netpol.Spec.Ingress).To(BeEmpty(), "Default NetworkPolicy should deny all ingress")
				g.Expect(netpol.Spec.Egress).To(BeEmpty(), "Default NetworkPolicy should deny all egress")
			}, 30*time.Second, time.Second).Should(Succeed())

			By("Verifying the Kubeconfig Secret is created")
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-kubeconfig"}, secret)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get Kubeconfig Secret")
				g.Expect(secret.Data).To(HaveKey("kubeconfig"))
				g.Expect(secret.Type).To(Equal(corev1.SecretTypeOpaque))
			}, 30*time.Second, time.Second).Should(Succeed())

			By("updating the ResourceQuota in Userconfig")
			updatedUserConfig := &myoperatorv1alpha1.UserConfig{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to get UserConfig resource for resourcequota update")
			updatedUserConfig.Spec.ResourceQuotas = &myoperatorv1alpha1.ResourceQuota{
				Pods: "10",
				CPU:  "2",
			}
			err = k8sClient.Update(context.Background(), updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to update UserConfig resource for resourcequota update")

			updatedUserConfig = &myoperatorv1alpha1.UserConfig{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to get updated UserConfig resource")

			By("Verifying the resourcequota is updated or not")
			Eventually(func(g Gomega) {
				resourceQuota := &corev1.ResourceQuota{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "default-resource-quota"}, resourceQuota)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get updated ResourceQuota")
				g.Expect(resourceQuota.Spec.Hard).To(HaveKeyWithValue(corev1.ResourceName("pods"), EqualQuantity("10"))) // Updated value
				g.Expect(resourceQuota.Spec.Hard).To(HaveKeyWithValue(corev1.ResourceName("cpu"), EqualQuantity("2")))   // Updated value
			}, 120*time.Second, time.Second).Should(Succeed())

			updatedUserConfig = &myoperatorv1alpha1.UserConfig{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to get updated UserConfig resource")

			By("Updating the LimitRange in UserConfig")
			updatedUserConfig.Spec.LimitRange.Limits[0].Max.CPU = "2"        // Update Max CPU
			updatedUserConfig.Spec.LimitRange.Limits[0].Max.Memory = "2Gi"   // Update Max Memory
			updatedUserConfig.Spec.LimitRange.Limits[0].Min.CPU = "200m"     // Update Min CPU
			updatedUserConfig.Spec.LimitRange.Limits[0].Min.Memory = "256Mi" // Update Min Memory
			err = k8sClient.Update(context.Background(), updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to update UserConfig resource for limitrange update")

			By("Verifying the LimitRange is updated or not")
			Eventually(func(g Gomega) {
				limitRange := &corev1.LimitRange{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-limit-range"}, limitRange)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get updated LimitRange")
				g.Expect(limitRange.Spec.Limits).To(HaveLen(1))
				limit := limitRange.Spec.Limits[0]
				g.Expect(limit.Type).To(Equal(corev1.LimitTypeContainer))
				g.Expect(limit.Max.Cpu().String()).To(Equal("2"))        // Updated value
				g.Expect(limit.Max.Memory().String()).To(Equal("2Gi"))   // Updated value
				g.Expect(limit.Min.Cpu().String()).To(Equal("200m"))     // Updated value
				g.Expect(limit.Min.Memory().String()).To(Equal("256Mi")) // Updated value
			}, 120*time.Second, time.Second).Should(Succeed())

			updatedUserConfig = &myoperatorv1alpha1.UserConfig{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{Name: "test-user"}, updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to get updated UserConfig resource")

			By("Updating the Role in UserConfig")
			updatedUserConfig.Spec.Permissions.Resources[0].Operation = "RUD" // Update Operation

			// Add a new permission for "deployments" with "CRUD"
			updatedUserConfig.Spec.Permissions.Resources = append(updatedUserConfig.Spec.Permissions.Resources, myoperatorv1alpha1.ResourcePermission{
				Resource:  "deployment",
				Operation: "CRUD",
			})
			err = k8sClient.Update(context.Background(), updatedUserConfig)
			Expect(err).NotTo(HaveOccurred(), "Failed to update UserConfig resource for role update")

			By("Verifying the Role is updated or not")
			Eventually(func(g Gomega) {
				role := &rbacv1.Role{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: userConfigNamespace, Name: "test-user-role"}, role)
				g.Expect(err).NotTo(HaveOccurred(), "Failed to get updated Role")
				g.Expect(role.Rules).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Resources": ContainElement("deployments"), // Updated resource
					"Verbs":     ContainElements("create", "get", "list", "watch", "update", "patch", "delete"),
				})))

				g.Expect(role.Rules).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Resources": ContainElement("pods"), // Updated resource
					"Verbs":     ContainElements("delete", "get", "list", "watch", "update", "patch"),
				})))

				g.Expect(role.Rules).To(ContainElement(MatchFields(IgnoreExtras, Fields{
					"Resources": ContainElement("pods"), // Updated resource
					"Verbs":     Not(ContainElements("create")),
				})))

			}, 30*time.Second, time.Second).Should(Succeed())
		})
	})
})

// Helper function to compare resource quantities
func EqualQuantity(expected string) types.GomegaMatcher {
	return WithTransform(func(q resource.Quantity) string { return q.String() }, Equal(expected))
}

// serviceAccountToken returns a token for the specified service account in the given namespace.
func serviceAccountToken() (string, error) {
	const tokenRequestRawString = `{
		"apiVersion": "authentication.k8s.io/v1",
		"kind": "TokenRequest"
	}`

	secretName := fmt.Sprintf("%s-token-request", serviceAccountName)
	tokenRequestFile := filepath.Join("/tmp", secretName)
	err := os.WriteFile(tokenRequestFile, []byte(tokenRequestRawString), os.FileMode(0o644))
	if err != nil {
		return "", err
	}

	var out string
	verifyTokenCreation := func(g Gomega) {
		cmd := exec.Command("kubectl", "create", "--raw", fmt.Sprintf(
			"/api/v1/namespaces/%s/serviceaccounts/%s/token",
			namespace,
			serviceAccountName,
		), "-f", tokenRequestFile)

		output, err := cmd.CombinedOutput()
		g.Expect(err).NotTo(HaveOccurred())

		var token tokenRequest
		err = json.Unmarshal([]byte(output), &token)
		g.Expect(err).NotTo(HaveOccurred())

		out = token.Status.Token
	}
	Eventually(verifyTokenCreation).Should(Succeed())

	return out, err
}

// getMetricsOutput retrieves and returns the logs from the curl pod used to access the metrics endpoint.
func getMetricsOutput() string {
	By("getting the curl-metrics logs")
	cmd := exec.Command("kubectl", "logs", "curl-metrics", "-n", namespace)
	metricsOutput, err := utils.Run(cmd)
	Expect(err).NotTo(HaveOccurred(), "Failed to retrieve logs from curl pod")
	Expect(metricsOutput).To(ContainSubstring("< HTTP/1.1 200 OK"))
	return metricsOutput
}

// tokenRequest is a simplified representation of the Kubernetes TokenRequest API response.
type tokenRequest struct {
	Status struct {
		Token string `json:"token"`
	} `json:"status"`
}
