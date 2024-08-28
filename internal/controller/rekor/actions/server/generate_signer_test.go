package server

import (
	"context"

	. "github.com/onsi/gomega"
	"github.com/securesign/operator/internal/controller/common/action"
	"github.com/securesign/operator/internal/controller/rekor/actions"

	"reflect"
	"testing"

	rhtasv1alpha1 "github.com/securesign/operator/api/v1alpha1"
	"github.com/securesign/operator/internal/controller/constants"
	testAction "github.com/securesign/operator/internal/testing/action"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGenerateSigner_CanHandle(t *testing.T) {
	tests := []struct {
		name         string
		status       []metav1.Condition
		canHandle    bool
		signer       rhtasv1alpha1.RekorSigner
		statusSigner rhtasv1alpha1.RekorSigner
	}{
		{
			name: "spec.signer.keyRef is not nil and status.signer.keyRef is nil",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: true,
			signer: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
		},
		{
			name: "spec.signer.keyRef is nil and status.signer.keyRef is not nil",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: false,
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
		},
		{
			name: "spec.signer.keyRef is nil and status.signer.keyRef is nil",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: true,
		},
		{
			name: "spec.signer.keyRef != status.signer.keyRef",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: true,
			signer: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "new_secret"}, Key: "private"},
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "old_secret"}, Key: "private"},
			},
		},
		{
			name: "spec.signer.keyRef == status.signer.keyRef",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: false,
			signer: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
		},
		{
			name: "spec.signer.passwordRef == status.signer.passwordRef",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: false,
			signer: rhtasv1alpha1.RekorSigner{
				KeyRef:      &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
				PasswordRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "password"},
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef:      &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
				PasswordRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "password"},
			},
		},
		{
			name: "spec.signer.passwordRef != status.signer.passwordRef",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: true, signer: rhtasv1alpha1.RekorSigner{
				KeyRef:      &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "new_secret"}, Key: "private"},
				PasswordRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "new_secret"}, Key: "password"},
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef:      &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "old_secret"}, Key: "private"},
				PasswordRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "old_secret"}, Key: "password"},
			},
		},
		{
			name: "spec.signer.kms != status.signer.kms",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: true,
			signer: rhtasv1alpha1.RekorSigner{
				KMS: "azurekeyvault://mykeyvaultname.vault.azure.net/keys/mykeyname",
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KMS: "awskms://1234abcd-12ab-34cd-56ef-1234567890ab?region=us-east-1",
			},
		},
		{
			name: "spec.signer.kms == status.signer.kms",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: false,
			signer: rhtasv1alpha1.RekorSigner{
				KMS: "awskms://1234abcd-12ab-34cd-56ef-1234567890ab?region=us-east-1",
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KMS: "awskms://1234abcd-12ab-34cd-56ef-1234567890ab?region=us-east-1",
			},
		},
		{
			name:      "no phase condition",
			status:    []metav1.Condition{},
			canHandle: true,
		},
		{
			name: "ConditionFalse",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionFalse,
					Reason: constants.Pending,
				},
			},
			canHandle: true,
		},
		{
			name: "ConditionTrue",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionTrue,
					Reason: constants.Ready,
				},
			},
			canHandle: false,
			signer: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
			statusSigner: rhtasv1alpha1.RekorSigner{
				KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
			},
		},
		{
			name: "ConditionUnknown",
			status: []metav1.Condition{
				{
					Type:   actions.SignerCondition,
					Status: metav1.ConditionUnknown,
					Reason: constants.Ready,
				},
			},
			canHandle: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := testAction.FakeClientBuilder().Build()
			a := testAction.PrepareAction(c, NewGenerateSignerAction())
			instance := rhtasv1alpha1.Rekor{
				Spec: rhtasv1alpha1.RekorSpec{
					Signer: tt.signer,
				},
				Status: rhtasv1alpha1.RekorStatus{
					Signer: tt.statusSigner,
				},
			}
			for _, status := range tt.status {
				meta.SetStatusCondition(&instance.Status.Conditions, status)
			}

			if got := a.CanHandle(context.TODO(), &instance); !reflect.DeepEqual(got, tt.canHandle) {
				t.Errorf("CanHandle() = %v, want %v", got, tt.canHandle)
			}
		})
	}
}

func TestGenerateSigner_Handle(t *testing.T) {
	g := NewWithT(t)
	type env struct {
		spec   rhtasv1alpha1.RekorSigner
		status rhtasv1alpha1.RekorSigner
	}
	type want struct {
		result *action.Result
		verify func(Gomega, *rhtasv1alpha1.Rekor)
	}
	tests := []struct {
		name string
		env  env
		want want
	}{
		{
			name: "use spec.signer.keyRef",
			env: env{
				spec: rhtasv1alpha1.RekorSigner{
					KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
				},
				status: rhtasv1alpha1.RekorSigner{},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KeyRef).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KeyRef.Name).Should(Equal("secret"))
					g.Expect(instance.Status.Signer.KeyRef.Key).Should(Equal("private"))

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
		{
			name: "generate signer key",
			env: env{
				spec:   rhtasv1alpha1.RekorSigner{},
				status: rhtasv1alpha1.RekorSigner{},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KeyRef).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KeyRef.Name).Should(ContainSubstring("rekor-signer-rekor-"))

					g.Expect(instance.Status.Signer.PasswordRef).Should(BeNil())

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
		{
			name: "replace status.signer.keyRef from spec",
			env: env{
				spec: rhtasv1alpha1.RekorSigner{
					KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "new_secret"}, Key: "private"},
				},
				status: rhtasv1alpha1.RekorSigner{
					KeyRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "old_secret"}, Key: "private"},
				},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KeyRef).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KeyRef.Name).Should(Equal("new_secret"))

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
		{
			name: "use spec.signer.KMS",
			env: env{
				spec: rhtasv1alpha1.RekorSigner{
					KMS: "awskms://1234abcd-12ab-34cd-56ef-1234567890ab?region=us-east-1",
				},
				status: rhtasv1alpha1.RekorSigner{},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KMS).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KMS).Should(Equal("awskms://1234abcd-12ab-34cd-56ef-1234567890ab?region=us-east-1"))

					g.Expect(instance.Status.Signer.KeyRef).Should(BeNil())
					g.Expect(instance.Status.Signer.PasswordRef).Should(BeNil())

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
		{
			name: "replace status.signer.KMS from spec",
			env: env{
				spec: rhtasv1alpha1.RekorSigner{
					KMS: "new-kms",
				},
				status: rhtasv1alpha1.RekorSigner{
					KMS: "old-kms",
				},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KMS).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KMS).Should(Equal("new-kms"))

					g.Expect(instance.Status.Signer.KeyRef).Should(BeNil())
					g.Expect(instance.Status.Signer.PasswordRef).Should(BeNil())

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
		{
			name: "spec with encrypted private key",
			env: env{
				spec: rhtasv1alpha1.RekorSigner{
					KeyRef:      &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "private"},
					PasswordRef: &rhtasv1alpha1.SecretKeySelector{LocalObjectReference: rhtasv1alpha1.LocalObjectReference{Name: "secret"}, Key: "password"},
				},
				status: rhtasv1alpha1.RekorSigner{},
			},
			want: want{
				result: testAction.StatusUpdate(),
				verify: func(g Gomega, instance *rhtasv1alpha1.Rekor) {
					g.Expect(instance.Status.Signer.KeyRef).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.KeyRef.Name).Should(Equal("secret"))
					g.Expect(instance.Status.Signer.KeyRef.Key).Should(Equal("private"))

					g.Expect(instance.Status.Signer.PasswordRef).ShouldNot(BeNil())
					g.Expect(instance.Status.Signer.PasswordRef.Name).Should(Equal("secret"))
					g.Expect(instance.Status.Signer.PasswordRef.Key).Should(Equal("password"))

					g.Expect(meta.IsStatusConditionTrue(instance.Status.Conditions, actions.SignerCondition)).Should(BeTrue())
					g.Expect(meta.IsStatusConditionFalse(instance.Status.Conditions, actions.ServerCondition)).Should(BeTrue())
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.TODO()
			instance := &rhtasv1alpha1.Rekor{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "rekor",
					Namespace: "default",
				},
				Spec: rhtasv1alpha1.RekorSpec{
					Signer: tt.env.spec,
				},
				Status: rhtasv1alpha1.RekorStatus{
					Signer: tt.env.status,
				},
			}

			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:   constants.Ready,
				Reason: constants.Pending,
			})

			meta.SetStatusCondition(&instance.Status.Conditions, metav1.Condition{
				Type:   actions.SignerCondition,
				Status: metav1.ConditionFalse,
				Reason: constants.Pending,
			})

			c := testAction.FakeClientBuilder().
				WithObjects(instance).
				WithStatusSubresource(instance).
				Build()

			a := testAction.PrepareAction(c, NewGenerateSignerAction())

			if got := a.Handle(ctx, instance); !reflect.DeepEqual(got, tt.want.result) {
				t.Errorf("CanHandle() = %v, want %v", got, tt.want.result)
			}
			if tt.want.verify != nil {
				tt.want.verify(g, instance)
			}
		})
	}
}
