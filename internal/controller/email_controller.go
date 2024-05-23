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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	emailv1 "email-operator/api/v1"
	"email-operator/internal/thirdparty"
)

// EmailReconciler reconciles a Email object
type EmailReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=email.example.com,resources=emails,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=email.example.com,resources=emails/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=email.example.com,resources=emails/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coordination.k8s.io,resources=leases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Email object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *EmailReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	// Fetch the Email instance
	email := &emailv1.Email{}
	err := r.Get(ctx, req.NamespacedName, email)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Log the creation or update action
	if email.Generation == 1 {
		logger.Info("Created new Email", "Email", email.Name)
	} else {
		logger.Info("Updated existing Email", "Email", email.Name)
	}

	// Fetch the EmailSenderConfig instance
	emailSenderConfig := &emailv1.EmailSenderConfig{}
	err = r.Get(ctx, types.NamespacedName{Name: email.Spec.SenderConfigRef, Namespace: req.Namespace}, emailSenderConfig)
	if err != nil {
		email.Status.DeliveryStatus = "Failed"
		if errors.IsNotFound(err) {
			email.Status.Error = fmt.Sprintf("EmailSenderConfig %s not found", email.Spec.SenderConfigRef)
		} else {
			email.Status.Error = fmt.Sprintf("Failed to get EmailSenderConfig: %v", err)
		}
		if updateErr := r.Status().Update(ctx, email); updateErr != nil {
			logger.Error(updateErr, "Failed to update Email status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	// Retrieve the API token from the secret
	secret := &corev1.Secret{}
	err = r.Get(ctx, types.NamespacedName{Name: emailSenderConfig.Spec.APITokenSecretRef, Namespace: req.Namespace}, secret)
	if err != nil {
		email.Status.DeliveryStatus = "Failed"
		email.Status.Error = fmt.Sprintf("Failed to get secret: %v", err)
		if updateErr := r.Status().Update(ctx, email); updateErr != nil {
			logger.Error(updateErr, "Failed to update Email status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}

	res, err := (&thirdparty.MailClient{
		Provider:  emailSenderConfig.Spec.Provider,
		ApiToken:  string(secret.Data["apiToken"]),
		Recipient: email.Spec.RecipientEmail,
		Subject:   email.Spec.Subject,
		From:      emailSenderConfig.Spec.SenderEmail,
		Text:      email.Spec.Body,
		Domain:    emailSenderConfig.Spec.Domain,
	}).SendEmail()

	if err != nil {
		email.Status.DeliveryStatus = "Failed"
		email.Status.Error = fmt.Sprintf("Failed to send email: %v", err)
		if updateErr := r.Status().Update(ctx, email); updateErr != nil {
			logger.Error(updateErr, "Failed to update Email status")
			return ctrl.Result{}, updateErr
		}
		return ctrl.Result{}, err
	}
	email.Status.DeliveryStatus = res.DeliveryStatus
	email.Status.Error = ""
	email.Status.MessageID = res.MessageID

	logger.Info("Email sent successfully",
		"MessageID", email.Status.MessageID,
		"DeliveryStatus", res.DeliveryStatus,
		"Subject", email.Spec.Subject,
		"From", emailSenderConfig.Spec.SenderEmail,
		"To", email.Spec.RecipientEmail,
	)

	if updateErr := r.Status().Update(ctx, email); updateErr != nil {
		logger.Error(updateErr, "Failed to update Email status")
		return ctrl.Result{}, updateErr
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EmailReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&emailv1.Email{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(r)
}
