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

package ctlog

import (
	"context"
	"time"

	"github.com/securesign/operator/api/v1alpha1"
	"github.com/securesign/operator/controllers/common/utils/kubernetes"
	"github.com/securesign/operator/controllers/constants"
	"github.com/securesign/operator/controllers/ctlog/actions"
	fulcio "github.com/securesign/operator/controllers/fulcio/actions"
	trillian "github.com/securesign/operator/controllers/trillian/actions"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("CTlog controller", func() {
	Context("CTlog controller test", func() {

		const (
			Name      = "test"
			Namespace = "default"
		)

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      Name,
				Namespace: Namespace,
			},
		}

		typeNamespaceName := types.NamespacedName{Name: Name, Namespace: Namespace}
		instance := &v1alpha1.CTlog{}

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).To(Not(HaveOccurred()))
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind CTlog")
			found := &v1alpha1.CTlog{}
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

		It("should successfully reconcile a custom resource for CTlog", func() {
			By("creating the custom resource for the Kind CTlog")
			err := k8sClient.Get(ctx, typeNamespaceName, instance)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				ptr := int64(1)
				instance := &v1alpha1.CTlog{
					ObjectMeta: metav1.ObjectMeta{
						Name:      Name,
						Namespace: Namespace,
					},

					Spec: v1alpha1.CTlogSpec{
						TreeID: &ptr,
					},
				}
				err = k8sClient.Create(ctx, instance)
				Expect(err).To(Not(HaveOccurred()))

			}

			By("Checking if the custom resource was successfully created")
			Eventually(func() error {
				found := &v1alpha1.CTlog{}
				return k8sClient.Get(ctx, typeNamespaceName, found)
			}, time.Minute, time.Second).Should(Succeed())

			By("Status conditions are initialized")
			Eventually(func() bool {
				found := &v1alpha1.CTlog{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.IsStatusConditionPresentAndEqual(found.Status.Conditions, constants.Ready, metav1.ConditionFalse)
			}, time.Minute, time.Second).Should(BeTrue())

			By("Pending phase until Trillian svc is created")
			Eventually(func() string {
				found := &v1alpha1.CTlog{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.FindStatusCondition(found.Status.Conditions, constants.Ready).Reason
			}, time.Minute, time.Second).Should(Equal(constants.Pending))

			By("Creating trillian service")
			Expect(k8sClient.Create(ctx, kubernetes.CreateService(Namespace, trillian.LogserverDeploymentName, 8091, constants.LabelsForComponent(trillian.LogServerComponentName, instance.Name)))).To(Succeed())
			Eventually(func() string {
				found := &v1alpha1.CTlog{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.FindStatusCondition(found.Status.Conditions, constants.Ready).Reason
			}, time.Minute, time.Second).Should(Equal(constants.Creating))

			By("Creating fulcio root cert")
			Expect(k8sClient.Create(ctx, kubernetes.CreateSecret("test", Namespace,
				map[string][]byte{"cert": []byte("fakeCert")},
				map[string]string{fulcio.FulcioCALabel: "cert"},
			))).To(Succeed())

			Eventually(func() string {
				found := &v1alpha1.CTlog{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.FindStatusCondition(found.Status.Conditions, constants.Ready).Reason
			}, time.Minute, time.Second).Should(Equal(constants.Creating))

			By("Key Secret is created")
			found := &v1alpha1.CTlog{}
			Eventually(func() *v1alpha1.SecretKeySelector {
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return found.Status.PrivateKeyRef
			}, time.Minute, time.Second).Should(Not(BeNil()))
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: found.Status.PrivateKeyRef.Name, Namespace: Namespace}, &corev1.Secret{})
			}, time.Minute, time.Second).Should(Not(HaveOccurred()))

			deployment := &appsv1.Deployment{}
			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, deployment)
			}, time.Minute, time.Second).Should(Succeed())

			By("Checking if Service was successfully created in the reconciliation")
			service := &corev1.Service{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: actions.ComponentName, Namespace: Namespace}, service)
			}, time.Minute, time.Second).Should(Succeed())
			Expect(service.Spec.Ports[0].Port).Should(Equal(int32(6963)))
			Expect(service.Spec.Ports[1].Port).Should(Equal(int32(80)))

			By("Move to Ready phase")
			// Workaround to succeed condition for Ready phase
			deployment.Status.Conditions = []appsv1.DeploymentCondition{
				{Status: corev1.ConditionTrue, Type: appsv1.DeploymentAvailable, Reason: constants.Ready}}
			Expect(k8sClient.Status().Update(ctx, deployment)).Should(Succeed())

			By("Waiting until CTlog instance is Ready")
			Eventually(func() bool {
				found := &v1alpha1.CTlog{}
				Expect(k8sClient.Get(ctx, typeNamespaceName, found)).Should(Succeed())
				return meta.IsStatusConditionTrue(found.Status.Conditions, constants.Ready)
			}, time.Minute, time.Second).Should(BeTrue())

			By("Checking if controller will return deployment to desired state")
			deployment = &appsv1.Deployment{}
			Eventually(func() error {
				return k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, deployment)
			}, time.Minute, time.Second).Should(Succeed())
			replicas := int32(99)
			deployment.Spec.Replicas = &replicas
			Expect(k8sClient.Status().Update(ctx, deployment)).Should(Succeed())
			Eventually(func() int32 {
				deployment = &appsv1.Deployment{}
				Expect(k8sClient.Get(ctx, types.NamespacedName{Name: actions.DeploymentName, Namespace: Namespace}, deployment)).Should(Succeed())
				return *deployment.Spec.Replicas
			}, time.Minute, time.Second).Should(Equal(int32(1)))
		})
	})
})
