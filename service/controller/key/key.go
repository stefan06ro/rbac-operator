package key

import (
	"github.com/giantswarm/microerror"
	corev1 "k8s.io/api/core/v1"
)

func ToNamespace(v interface{}) (corev1.Namespace, error) {
	if v == nil {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Namespace{}, v)
	}

	p, ok := v.(*corev1.Namespace)
	if !ok {
		return corev1.Namespace{}, microerror.Maskf(wrongTypeError, "expected '%T', got '%T'", &corev1.Namespace{}, v)
	}

	c := p.DeepCopy()

	return *c, nil
}
