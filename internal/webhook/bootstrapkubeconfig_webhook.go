// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package webhook

import (
	"context"
	b64 "encoding/base64"
	"encoding/pem"
	"fmt"
	// "fmt"
	infrav1 "github.com/vmware-tanzu/cluster-api-provider-bringyourownhost/apis/infrastructure/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// log is for logging in this package.
var bootstrapkubeconfiglog = logf.Log.WithName("bootstrapkubeconfig-resource")

// APIServerURLScheme is the url scheme for the APIServer
const APIServerURLScheme = "https"

type BootstrapKubeconfig struct{}

func (r *BootstrapKubeconfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&infrav1.BootstrapKubeconfig{}).
		WithValidator(r).
		Complete()
}

//+kubebuilder:webhook:path=/validate-infrastructure-cluster-x-k8s-io-v1beta1-bootstrapkubeconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=infrastructure.cluster.x-k8s.io,resources=bootstrapkubeconfigs,verbs=create;update,versions=v1beta1,name=vbootstrapkubeconfig.kb.io,admissionReviewVersions=v1

var _ webhook.CustomValidator = &BootstrapKubeconfig{}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *BootstrapKubeconfig) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	newBootstrapKubeconfig, ok := newObj.(*infrav1.BootstrapKubeconfig)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a BootstrapKubeconfig but got %T", newObj))
	}

	bootstrapkubeconfiglog.Info("validate update", "name", newBootstrapKubeconfig.Name)

	if err := validateAPIServer(newBootstrapKubeconfig); err != nil {
		return nil, err
	}

	if err := validateCAData(newBootstrapKubeconfig); err != nil {
		return nil, err
	}

	return nil, nil
}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *BootstrapKubeconfig) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {

	newBootstrapKubeconfig, ok := obj.(*infrav1.BootstrapKubeconfig)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a BootstrapKubeconfig but got %T", obj))
	}

	bootstrapkubeconfiglog.Info("validate create", "name", newBootstrapKubeconfig.Name)

	if err := validateAPIServer(newBootstrapKubeconfig); err != nil {
		return nil, err
	}

	if err := validateCAData(newBootstrapKubeconfig); err != nil {
		return nil, nil
	}

	return nil, nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *BootstrapKubeconfig) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	deletedBootstrapKubeconfig, ok := obj.(*infrav1.BootstrapKubeconfig)
	if !ok {
		return nil, apierrors.NewBadRequest(fmt.Sprintf("expected a BootstrapKubeconfig but got %T", obj))
	}
	bootstrapkubeconfiglog.Info("validate delete", "name", deletedBootstrapKubeconfig.Name)

	return nil, nil
}

func validateAPIServer(r *infrav1.BootstrapKubeconfig) error {
	if r.Spec.APIServer == "" {
		return field.Invalid(field.NewPath("spec").Child("apiserver"), r.Spec.APIServer, "APIServer field cannot be empty")
	}

	parsedURL, err := url.Parse(r.Spec.APIServer)
	if err != nil {
		return field.Invalid(field.NewPath("spec").Child("apiserver"), r.Spec.APIServer, "APIServer URL is not valid")
	}
	if !isURLValid(parsedURL) {
		return field.Invalid(field.NewPath("spec").Child("apiserver"), r.Spec.APIServer, "APIServer is not of the format https://hostname:port")
	}
	return nil
}

func validateCAData(r *infrav1.BootstrapKubeconfig) error {
	if r.Spec.CertificateAuthorityData == "" {
		return field.Invalid(field.NewPath("spec").Child("caData"), r.Spec.CertificateAuthorityData, "CertificateAuthorityData field cannot be empty")
	}

	decodedCAData, err := b64.StdEncoding.DecodeString(r.Spec.CertificateAuthorityData)
	if err != nil {
		return field.Invalid(field.NewPath("spec").Child("caData"), r.Spec.CertificateAuthorityData, "cannot base64 decode CertificateAuthorityData")
	}

	block, _ := pem.Decode(decodedCAData)
	if block == nil {
		return field.Invalid(field.NewPath("spec").Child("caData"), r.Spec.CertificateAuthorityData, "CertificateAuthorityData is not PEM encoded")
	}

	return nil
}
func isURLValid(parsedURL *url.URL) bool {
	if parsedURL.Host == "" || parsedURL.Scheme != APIServerURLScheme || parsedURL.Port() == "" {
		return false
	}
	return true
}
