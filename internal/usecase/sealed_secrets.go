package usecase

import (
	"context"
	"fmt"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"

	sealedsecretsv1alpha1 "github.com/bitnami-labs/sealed-secrets/pkg/apis/sealedsecrets/v1alpha1"

	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	myoperatorv1alpha1 "01cloud/zoperator/api/v1alpha1"
)

func (u *UserConfigUseCase) ReconcileSealedSecrets(ctx context.Context, uc *myoperatorv1alpha1.UserConfig) error {
	log := log.FromContext(ctx)

	namespace := &corev1.Namespace{}
	if err := u.Get(ctx, client.ObjectKey{Name: uc.Name}, namespace); err != nil {
		return fmt.Errorf("namespace %s not found: %w", uc.Name, err)
	}

	for _, secret := range uc.Spec.Secrets {
		if secret.Type != "sealed" || secret.SealedSecret == nil {
			continue
		}

		sealedSecret := &sealedsecretsv1alpha1.SealedSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secret.Name,
				Namespace: uc.Name,
			},
			Spec: sealedsecretsv1alpha1.SealedSecretSpec{
				EncryptedData: secret.SealedSecret.EncryptedData,
				Template: sealedsecretsv1alpha1.SecretTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secret.Name,
						Namespace: uc.Name,
					},
					Type: corev1.SecretTypeOpaque,
				},
			},
		}

		// Set ownership reference
		if err := controllerutil.SetControllerReference(uc, sealedSecret, u.Scheme); err != nil {
			return fmt.Errorf("failed to set controller reference for SealedSecret %s: %w", secret.Name, err)
		}

		// Create or update the SealedSecret
		existing := &sealedsecretsv1alpha1.SealedSecret{}
		if err := u.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: uc.Name}, existing); err != nil {
			if apierrors.IsNotFound(err) {
				if err := u.Create(ctx, sealedSecret); err != nil {
					return fmt.Errorf("failed to create SealedSecret %s: %w", secret.Name, err)
				}
				log.Info("Created SealedSecret", "name", secret.Name)
			} else {
				return fmt.Errorf("failed to get existing SealedSecret %s: %w", secret.Name, err)
			}
		} else {
			existing.Spec = sealedSecret.Spec
			if err := u.Update(ctx, existing); err != nil {
				return fmt.Errorf("failed to update SealedSecret %s: %w", secret.Name, err)
			}
			log.Info("Updated SealedSecret", "name", secret.Name)
		}
	}

	return nil
}
