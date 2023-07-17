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

package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"k8s.io/apimachinery/pkg/types"

	performancev1alpha1 "github.com/le0nard01/example/api/v1alpha1"
)

// TesterReconciler reconciles a Tester object
type TesterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Permissões

//+kubebuilder:rbac:groups=performance.oste.com.br,resources=testers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=performance.oste.com.br,resources=testers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=performance.oste.com.br,resources=testers/finalizers,verbs=update

//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list; // Permissões para os Pods
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete // Permissões para os Jobs
//+kubebuilder:rbac:groups=core,resources=node,verbs=get;list;watch;update;patch // Permissões os Nodes

// Reconcile faz parte do loop principal de reconciliação do Kubernetes,
// que tem como objetivo aproximar o estado atual do cluster do estado desejado.
// TODO(usuario): Modificar a função Reconcile para comparar o estado especificado
// pelo objeto Tester com o estado atual do cluster e, em seguida,
// realizar operações para fazer com que o estado do cluster reflita o estado especificado
// pelo usuário.

// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *TesterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	tester := &performancev1alpha1.Tester{}
	err := r.Get(ctx, req.NamespacedName, tester)

	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Recurso não encontrado")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Failed to get Tester")
		return ctrl.Result{}, err
	}

	nodeList := &corev1.NodeList{}
	err = r.List(ctx, nodeList)
	if err != nil {
		logger.Error(err, "Erro ao listar os nodes")
		return ctrl.Result{}, err
	}

	nodeNames := []string{}
	for _, node := range nodeList.Items {
		nodeNames = append(nodeNames, node.Name)
	}

	nodesNotTested := difference(nodeNames, tester.Status.NodesTested)
	nodesTested := tester.Status.NodesTested
	if tester.Spec.Tests.IopsTest.Enabled {
		for _, nodeName := range nodesNotTested {
			jobName := types.NamespacedName{
				Name:      ("job-" + tester.Name + nodeName),
				Namespace: tester.Namespace,
			}

			err := r.Get(ctx, jobName, &batchv1.Job{})

			if err != nil && apierrors.IsNotFound(err) {
				logger.Info("Criando job: " + jobName.Name)
				job := r.jobCreate(jobName, nodeName, tester)
				err = r.Create(ctx, job)
				if err != nil {
					logger.Error(err, "Falha ao criar o Job: "+jobName.Name)
				} else {
					failed, err := r.waitJob(ctx, job, (time.Second * 60))
					if err != nil {
						logger.Error(err, "Falha ao aguardar finalização do Job: "+jobName.Name)
						return ctrl.Result{}, err
					}
					if failed {
						err := r.unschedulableNode(ctx, nodeName)
						if err != nil {
							logger.Error(err, "Falha deixar node como Unschedulable: "+nodeName)
							return ctrl.Result{}, err
						}
						logger.Info("Desabilitando node: " + nodeName)
					}
					nodesTested = append(nodesTested, nodeName)
				}

			}
		}
		if !reflect.DeepEqual(nodesTested, tester.Status.NodesTested) {
			tester.Status.NodesTested = nodesTested
			err := r.Status().Update(ctx, tester)
			if err != nil {
				logger.Error(err, "Falha ao atualizar Status do Tester")
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{}, nil
}

// jobCreate cria a estrutura do Job baseado nas informações do Tester (namespace e name),
// também no nome do node para força-lo a subir no node específico.
func (r *TesterReconciler) jobCreate(jobName types.NamespacedName, nodeName string, tester *performancev1alpha1.Tester) *batchv1.Job {
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName.Name,
			Namespace: jobName.Namespace,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					NodeName: nodeName,
					Containers: []corev1.Container{
						{
							Name:  "iops-test",
							Image: "quay.io/loste/fiotest-operator:latest",
							Env: []corev1.EnvVar{
								{
									Name:  "readThreshold",
									Value: fmt.Sprint(tester.Spec.Tests.IopsTest.ReadThreshold),
								},
								{
									Name:  "writeThreshold",
									Value: fmt.Sprint(tester.Spec.Tests.IopsTest.WriteThreshold),
								},
							},
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
				},
			},
			BackoffLimit: new(int32),
		},
	}
	*job.Spec.BackoffLimit = 0
	ctrl.SetControllerReference(tester, job, r.Scheme)
	return job
}

// waitJob aguarda o término do job e verifica se ele falhou ou deu sucesso.
// Retorna true se o job tiver falhado.
func (r *TesterReconciler) waitJob(ctx context.Context, job *batchv1.Job, timeout time.Duration) (bool, error) {
	jobKey := client.ObjectKeyFromObject(job)

	// Aguarda a criação do objeto job
	if err := r.Get(ctx, jobKey, job); err != nil {
		return false, err
	}
	// Aguarda o pod do job ser criado
	podList := &corev1.PodList{}
	err := r.List(ctx, podList, client.InNamespace(job.Namespace), client.MatchingLabels{"job-name": job.Name})
	if err != nil {
		return false, err
	}
	for len(podList.Items) == 0 {
		time.Sleep(1 * time.Second)
		err = r.List(ctx, podList, client.InNamespace(job.Namespace), client.MatchingLabels{"job-name": job.Name})
		if err != nil {
			return false, err
		}
	}

	// Aguarda a conclusão do job ou o tempo limite
	deadline := time.Now().Add(timeout)
	for {
		if time.Now().After(deadline) {
			break
		}

		err := r.Get(ctx, jobKey, job)
		if err != nil {
			return false, err
		}

		if job.Status.Failed > 0 {
			return job.Status.Failed > 0, nil
		} else if job.Status.Succeeded > 0 {
			return false, nil
		}
		time.Sleep(1 * time.Second)
	}
	return true, nil
}

// Faz o "Update" do Node e ativar a opção spec.unschedulable
// para torna-lo inacessível para novos pods.
func (r *TesterReconciler) unschedulableNode(ctx context.Context, nodeName string) error {
	logger := log.FromContext(ctx)

	node := &corev1.Node{}
	err := r.Get(ctx, types.NamespacedName{Name: nodeName}, node)
	if err != nil {
		return err
	}

	// Verifica se o nó já está marcado como ScheduleDisabled
	if node.Spec.Unschedulable {
		return nil
	}
	// Adiciona a taint ao nó para marcar como ScheduleDisabled
	node.Spec.Unschedulable = true
	logger.Info("Debug 4")
	err = r.Update(ctx, node)
	if err != nil {
		return err
	}

	if err != nil {
		return err
	}

	return nil
}

// Função básica de diferença.
func difference(a, b []string) []string {
	mb := make(map[string]struct{}, len(b))
	for _, x := range b {
		mb[x] = struct{}{}
	}
	var diff []string
	for _, x := range a {
		if _, found := mb[x]; !found {
			diff = append(diff, x)
		}
	}
	return diff
}

// SetupWithManager sets up the controller with the Manager.
func (r *TesterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&performancev1alpha1.Tester{}).
		Complete(r)
}
