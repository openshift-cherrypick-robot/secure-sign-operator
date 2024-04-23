/*
Copyright 2023.

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

package fulcio

import (
	"context"
	"time"

	"github.com/securesign/operator/api/v1alpha1"
	"github.com/securesign/operator/controllers/common/utils/kubernetes"
	"github.com/securesign/operator/controllers/constants"
	"github.com/securesign/operator/controllers/fulcio/actions"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("Fulcio hot update", func() {
	Context("Fulcio hot update test", func() {

		const (
			Name      = "test-fulcio"
			Namespace = "update"
		)

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: Namespace,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: Name, Namespace: Namespace}
		instance := &v1alpha1.Fulcio{}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind Fulcio")
			found := &v1alpha1.Fulcio{}
			err := k8sClient.Get(ctx, typeNamespaceName, found)
			Expect(err).To(Not(HaveOccurred()))

			Eventually(func() error {
				return k8sClient.Delete(context.TODO(), found)
			}, 2*time.Minute, time.Second).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)
		})

		It("should successfully reconcile a custom resource for Fulcio", func() {
			By("creating the custom resource for the Kind Fulcio")
			err := k8sClient.Get(ctx, typeNamespaceName, instance)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				instance := &v1alpha1.Fulcio{
					ObjectMeta: metav1.ObjectMeta{
						Name:      Name,
						Namespace: Namespace,
					},
					Spec: v1alpha1.FulcioSpec{
						ExternalAccess: v1alpha1.ExternalAccess{
							Host:    "fulcio.localhost",
							Enabled: true,
						},
						Config: v1alpha1.FulcioConfig{
							OIDCIssuers: []v1alpha1.OIDCIssuer{
								{
									IssuerURL: "test",
									Issuer:    "test",
									ClientID:  "test",
									Type:      "email",
								},
							},
						},
						Certificate: v1alpha1.FulcioCert{
							OrganizationName:  "MyOrg",
							OrganizationEmail: "my@email.com",
							CommonName:        "local",
						},
						Monitoring: v1alpha1.MonitoringConfig{Enabled: false},
					},
				}
				err = k8sClient.Create(ctx, instance)
				Expect(err).To(Not(HaveOccurred()))
			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &v1alpha1.Fulcio{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			deployment := &appsv1.Deployment{}
			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, deployment)
			}, time.Minute, time.Second).Should(Succeed())

			By("Move to Ready phase")
			// Workaround to succeed condition for Ready phase
			deployment.Status.Conditions = []appsv1.DeploymentCondition{
				{Status: corev1.ConditionTrue, Type: appsv1.DeploymentAvailable, Reason: constants.Ready}}
			Expect(k8sClient.Status().Update(ctx, deployment)).Should(Succeed())

			By("Waiting until Fulcio instance is Ready")
			found := &v1alpha1.Fulcio{}
			Eventually(func() bool {
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.IsStatusConditionTrue(found.Status.Conditions, constants.Ready)
			}, time.Minute, time.Second).Should(BeTrue())

			By("Key rotation")
			Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
			found.Spec.Certificate.PrivateKeyPasswordRef = &v1alpha1.SecretKeySelector{
				LocalObjectReference: v1alpha1.LocalObjectReference{
					Name: "password-secret",
				},
				Key: "password",
			}
			Expect(k8sClient.Update(ctx, found)).To(Succeed())

			By("Pending phase until password key is resolved")
			Eventually(func() string {
				found := &v1alpha1.Fulcio{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.FindStatusCondition(found.Status.Conditions, constants.Ready).Reason
			}, time.Minute, time.Second).Should(Equal(constants.Pending))

			By("Creating password secret with cert password")
			Expect(k8sClient.Create(ctx, kubernetes.CreateSecret("password-secret", typeNamespaceName.Namespace, map[string][]byte{
				"password": []byte("secret"),
			}, constants.LabelsForComponent(actions.ComponentName, instance.Name)))).To(Succeed())

			By("Status field changed")
			Eventually(func() string {
				found := &v1alpha1.Fulcio{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return found.Status.Certificate.PrivateKeyPasswordRef.Name
			}, time.Minute, time.Second).Should(Equal("password-secret"))

			Eventually(func() bool {
				found := &v1alpha1.Fulcio{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.IsStatusConditionTrue(found.Status.Conditions, actions.CertCondition)
			}, time.Minute, time.Second).Should(BeTrue())

			By("Fulcio deployment is updated")
			Eventually(func() bool {
				updated := &appsv1.Deployment{}
				k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, updated)
				return equality.Semantic.DeepDerivative(deployment.Spec.Template.Spec.Volumes, updated.Spec.Template.Spec.Volumes)
			}, time.Minute, time.Second).Should(BeFalse())

			time.Sleep(10 * time.Second)

			By("Config update")
			Expect(k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, deployment)).To(Succeed())

			By("Update OIDC")
			Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
			found.Spec.Config.OIDCIssuers[0] = v1alpha1.OIDCIssuer{
				IssuerURL: "fake",
				Issuer:    "fake",
				ClientID:  "fake",
				Type:      "email",
			}
			Expect(k8sClient.Update(ctx, found)).To(Succeed())

			By("Fulcio deployment is updated")
			Eventually(func() bool {
				updated := &appsv1.Deployment{}
				k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, updated)
				return equality.Semantic.DeepDerivative(deployment.Spec.Template.Spec.Volumes, updated.Spec.Template.Spec.Volumes)
			}, time.Minute, time.Second).Should(BeFalse())
		})
	})
})
